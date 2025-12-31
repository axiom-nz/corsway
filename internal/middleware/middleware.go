package middleware

import (
	"log"
	"net/http"
	"slices"
	"sync"
	"time"

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

func limitRate(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	var (
		counts     = make(map[string]int)
		countsLock sync.Mutex
	)

	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		countsLock.Lock()
		count, exists := counts[ip]

		if !exists {
			counts[ip] = 1
			go func(ip string) {
				time.Sleep(cfg.RateLimitWindow)
				countsLock.Lock()
				delete(counts, ip)
				countsLock.Unlock()
			}(ip)
		} else if count >= cfg.RateLimit {
			countsLock.Unlock()
			log.Printf("Rate limit exceeded for %s", ip)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		} else {
			counts[ip]++
		}
		countsLock.Unlock()

		next(w, r)
	}
}
