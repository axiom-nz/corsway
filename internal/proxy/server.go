package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/axiom-nz/corsway/internal/config"
	"github.com/axiom-nz/corsway/internal/middleware"
)

type Server struct {
	Config *config.Config
	Client *http.Client
	Logger *log.Logger

	handler       http.Handler
	requestCounts map[string]int
	countsLock    sync.Mutex
}

func NewServer(cfg *config.Config, logger *log.Logger) *Server {
	client := &http.Client{
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

	server := &Server{
		Config:        cfg,
		Client:        client,
		Logger:        logger,
		requestCounts: make(map[string]int),
	}

	server.handler = middleware.Chain(cfg, server.handleProxy)

	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	setResponseCorsHeaders(w, r)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get URL from query path
	targetURL := strings.TrimPrefix(r.RequestURI, "/")
	if targetURL == "" {
		log.Printf("Missing URL in query path from %s", r.RemoteAddr)
		http.Error(w, "URL is required in query path", http.StatusBadRequest)
		return
	}

	// Prepare URL
	preparedURL, err := PrepareURL(targetURL)
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
	resp, err := s.Client.Do(req)
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
