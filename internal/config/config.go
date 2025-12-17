package config

import "time"

type Config struct {
	Port            int
	RateLimit       int
	RateLimitWindow time.Duration
	MaxRequestBytes int64
	OriginWhitelist []string
}
