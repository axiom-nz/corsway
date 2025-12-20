package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/axiom-nz/corsway/internal/config"
)

var (
	requestCounts = make(map[string]int)
	countsLock    = sync.Mutex{}
	client        = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives:     true,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			MaxConnsPerHost:       100,
		},
	}
)

func init() {
	// Register additional MIME types
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".json", "application/json")
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load config with defaults
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		log.Fatalf("Failed to load config: %v", err)
	}

	// Register handlers
	handlerStack := limitSources(cfg, limitRate(cfg, limitSize(cfg, handler)))

	log.Printf("Starting server:\n  Port: %d\n  Rate limit: %d\n  Window: %v\n  Max request size: %d\n  Origin Whitelist: %v",
		cfg.Port, cfg.RateLimit, cfg.RateLimitWindow, cfg.MaxRequestBytes, cfg.OriginWhitelist)

	// Start server
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), handlerStack))
}

func prepareURL(rawURL string) (string, error) {
	// Remove any whitespace
	rawURL = strings.TrimSpace(rawURL)

	// Fix common URL issues
	if strings.HasPrefix(rawURL, "//") {
		rawURL = "https:" + rawURL
	} else if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	// Fix double slashes in the path (except for the protocol)
	parts := strings.SplitN(rawURL, "://", 2)
	if len(parts) == 2 {
		protocol := parts[0]
		rest := strings.Replace(parts[1], "//", "/", -1)
		rawURL = protocol + "://" + rest
	}

	// Parse and validate the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	// Ensure the URL has a host
	if parsedURL.Host == "" {
		return "", fmt.Errorf("invalid URL: missing host")
	}

	return parsedURL.String(), nil
}

func setResponseCorsHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Expose-Headers", "*")

	if reqHeaders := r.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
		w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
	} else {
		w.Header().Set("Access-Control-Allow-Headers", "*")
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	setResponseCorsHeaders(w, r)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get URL from query path
	targetURL := strings.TrimPrefix(r.URL.RequestURI(), "/")
	if targetURL == "" {
		log.Printf("Missing URL in query path from %s", r.RemoteAddr)
		http.Error(w, "URL is required in query path", http.StatusBadRequest)
		return
	}

	// Prepare URL
	preparedURL, err := prepareURL(targetURL)
	if err != nil {
		log.Printf("Invalid URL %q from %s: %v", targetURL, r.RemoteAddr, err)
		http.Error(w, fmt.Sprintf("Invalid URL: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Proxying request to %q from %s", preparedURL, r.RemoteAddr)

	// Create request using same method as original
	req, err := http.NewRequestWithContext(r.Context(), r.Method, preparedURL, r.Body)
	if err != nil {
		log.Printf("Failed to create request for %q: %v", preparedURL, err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Set headers
	// todo: randomised user agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if r.Header.Get("Content-Type") != "" {
		req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to fetch %q: %v", preparedURL, err)
		if strings.Contains(err.Error(), "no such host") {
			http.Error(w, "Invalid host", http.StatusBadGateway)
		} else if strings.Contains(err.Error(), "timeout") {
			http.Error(w, "Request timed out", http.StatusGatewayTimeout)
		} else {
			http.Error(w, "Failed to fetch URL", http.StatusBadGateway)
		}
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Override response cors headers
	setResponseCorsHeaders(w, r)

	w.WriteHeader(resp.StatusCode)

	// Copy response body
	written, err := io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response for %q after %d bytes: %v", preparedURL, written, err)
		return
	}

	log.Printf("Proxied %d bytes from %q", written, preparedURL)
}

func limitRate(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

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
