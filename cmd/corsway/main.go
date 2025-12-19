package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/axiom-nz/corsway/internal/config"
)

var (
	cfg = config.Config{}

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

	flag.IntVar(&cfg.Port, "port", 8080, "Port to listen on")
	flag.IntVar(&cfg.RateLimit, "rate-limit", 20, "Maximum number of requests per rate-limit-window to allow")
	flag.DurationVar(&cfg.RateLimitWindow, "rate-limit-window", 5*60, "Duration of the rate-limit window")
	flag.Int64Var(&cfg.MaxRequestBytes, "max-request-bytes", 10<<20, "Maximum size of the request body in bytes")
	whitelist := flag.String("whitelist", "", "Comma-separated list of Origins to allow")
	flag.Parse()

	if strings.Trim(*whitelist, " ") != "" {
		cfg.OriginWhitelist = strings.Split(*whitelist, ",")
	}

	// Register handlers
	handlerStack := limitSources(limitRate(limitSize(handler)))

	// Start server
	log.Println("Starting server on port", cfg.Port)
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

func handler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	// Handle OPTIONS method
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

	// Create request
	req, err := http.NewRequestWithContext(r.Context(), r.Method, preparedURL, nil)
	if err != nil {
		log.Printf("Failed to create request for %q: %v", preparedURL, err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Set headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

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

	// Ensure CORS headers are set
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	written, err := io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response for %q after %d bytes: %v", preparedURL, written, err)
		return
	}

	log.Printf("Successfully proxied %d bytes from %q", written, preparedURL)
}

func limitRate(next http.HandlerFunc) http.HandlerFunc {
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

func limitSize(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxRequestBytes)
		next(w, r)
	}
}

func limitSources(next http.HandlerFunc) http.HandlerFunc {
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
