package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/axiom-nz/corsway/internal/config"
	"github.com/axiom-nz/corsway/internal/proxy"
)

func init() {
	// Register additional MIME types
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".json", "application/json")
}

func main() {
	// Load config with defaults
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	logger.Printf("Starting server:\n  Port: %d\n  Rate limit: %d\n  Window: %v\n  Max request size: %d\n  Origin Whitelist: %v",
		cfg.Port, cfg.RateLimit, cfg.RateLimitWindow, cfg.MaxRequestBytes, cfg.OriginWhitelist)

	// Start server
	server := proxy.NewServer(cfg, logger)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), server))
}
