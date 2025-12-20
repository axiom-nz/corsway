package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port            int
	RateLimit       int
	RateLimitWindow time.Duration
	MaxRequestBytes int64
	OriginWhitelist []string
}

// Load initialises the application configuration by processing inputs from
// multiple sources. It applies the following order of precedence (highest to lowest):
//
//  1. Command-line flags (e.g., --port=9000)
//  2. Environment variables (e.g., PORT=9000)
//  3. Internal hardcoded defaults
func Load(args []string) (*Config, error) {
	cfg := Config{
		Port:            8080,
		RateLimit:       20,
		RateLimitWindow: 5 * time.Minute,
		MaxRequestBytes: 10 << 20,
	}

	var whitelist string

	fs := flag.NewFlagSet("corsway", flag.ContinueOnError)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "Port to listen on")
	fs.IntVar(&cfg.RateLimit, "rate-limit", cfg.RateLimit, "Maximum number of requests per rate-limit-window to allow")
	fs.DurationVar(&cfg.RateLimitWindow, "rate-limit-window", cfg.RateLimitWindow, "Duration of the rate-limit window")
	fs.Int64Var(&cfg.MaxRequestBytes, "max-request-bytes", cfg.MaxRequestBytes, "Maximum size of the request body in bytes")
	fs.StringVar(&whitelist, "whitelist", "", "Comma-separated list of Origins to allow")

	if portVal := os.Getenv("PORT"); portVal != "" {
		port, err := strconv.Atoi(portVal)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT environment variable: %v", err)
		}
		cfg.Port = port
	}

	if rateLimitVal := os.Getenv("RATE_LIMIT"); rateLimitVal != "" {
		rateLimit, err := strconv.Atoi(rateLimitVal)
		if err != nil {
			return nil, fmt.Errorf("invalid RATE_LIMIT environment variable: %v", err)
		}
		cfg.RateLimit = rateLimit
	}

	if rateLimitWindowVal := os.Getenv("RATE_LIMIT_WINDOW"); rateLimitWindowVal != "" {
		rateLimitWindow, err := time.ParseDuration(rateLimitWindowVal)
		if err != nil {
			return nil, fmt.Errorf("invalid RATE_LIMIT_WINDOW environment variable: %v", err)
		}
		cfg.RateLimitWindow = rateLimitWindow
	}

	if maxRequestBytesVal := os.Getenv("MAX_REQUEST_BYTES"); maxRequestBytesVal != "" {
		maxRequestBytes, err := strconv.ParseInt(maxRequestBytesVal, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid MAX_REQUEST_BYTES environment variable: %v", err)
		}
		cfg.MaxRequestBytes = maxRequestBytes
	}

	whitelist = os.Getenv("WHITELIST")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if trimmed := strings.TrimSpace(whitelist); trimmed != "" {
		cfg.OriginWhitelist = strings.Split(trimmed, ",")
	} else {
		cfg.OriginWhitelist = []string{}
	}

	return &cfg, nil
}
