package middleware

import (
	"log"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/axiom-nz/corsway/internal/config"
)

var (
	requestCounts = make(map[string]int)
	countsLock    = sync.Mutex{}
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

func limitRate(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		countsLock.Lock()
		count, exists := requestCounts[ip]

		if !exists {
			requestCounts[ip] = 1
			go func(ip string) {
				time.Sleep(cfg.RateLimitWindow)
				countsLock.Lock()
				delete(requestCounts, ip)
				countsLock.Unlock()
			}(ip)
		} else if count >= cfg.RateLimit {
			countsLock.Unlock()
			log.Printf("Rate limit exceeded for %s", ip)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		} else {
			requestCounts[ip]++
		}
		countsLock.Unlock()

		next(w, r)
	}
}

// getClientIP returns the client IP address from the request headers
func getClientIP(r *http.Request) string {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		// If multiple proxies are used, take the first one
		// If cut fails to find a comma, return the entire string regardless
		source, _, _ := strings.Cut(forwardedFor, ",")
		return strings.TrimSpace(source)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
