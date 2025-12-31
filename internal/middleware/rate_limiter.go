package middleware

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/axiom-nz/corsway/internal/config"
	"golang.org/x/time/rate"
)

type clientData struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func limitRate(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	var (
		mu      sync.Mutex
		clients = make(map[string]*clientData)
	)

	// Run rate limiter cleanup outside of the http request path
	go func() {
		for range time.Tick(time.Minute) {
			mu.Lock()
			for ip, data := range clients {
				if time.Since(data.lastSeen) > 5*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(cfg, r)

		mu.Lock()
		client, exists := clients[ip]

		if !exists {
			client = &clientData{
				limiter:  rate.NewLimiter(rate.Every(cfg.RateLimitWindow), cfg.RateLimit),
				lastSeen: time.Now(),
			}
			clients[ip] = client
		}
		client.lastSeen = time.Now()
		allowed := client.limiter.Allow()
		mu.Unlock()

		if !allowed {
			log.Printf("Rate limit exceeded by %s", ip)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}
