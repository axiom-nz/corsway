package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
	handler := proxy.NewProxy(cfg, logger)
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Listen: %s\n", err)
		}
	}()

	// Handle termination & shutdown
	<-ctx.Done()
	stop()
	log.Printf("Shutting down server on port %d", cfg.Port)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
}
