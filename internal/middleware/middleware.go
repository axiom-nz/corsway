package middleware

import (
	"log"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/axiom-nz/corsway/internal/config"
)

// Chain applies rate limiting, origin whitelisting, and request size limits to the given handler
func Chain(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return limitRate(cfg, limitSources(cfg, limitSize(cfg, next)))
}

func limitSize(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxRequestBytes)
		next(w, r)
	}
}

func limitSources(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(cfg.OriginWhitelist) == 0 {
			next(w, r)
			return
		}

		if slices.Contains(cfg.OriginWhitelist, r.Header.Get("Origin")) {
			next(w, r)
			return
		}

		log.Printf("Blocked request from %s", r.RemoteAddr)
		http.Error(w, "Blocked request", http.StatusForbidden)
		return
	}
}

// getClientIP returns the client IP address from the request headers
// If the TrustProxy flag is set, it will first check the X-Forwarded-For header
func getClientIP(cfg *config.Config, r *http.Request) string {
	if cfg.TrustProxy {
		forwardedFor := r.Header.Get("X-Forwarded-For")
		if forwardedFor != "" {
			// If multiple proxies are used, take the first one
			// If cut fails to find a comma, return the entire string regardless
			source, _, _ := strings.Cut(forwardedFor, ",")
			return strings.TrimSpace(source)
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
