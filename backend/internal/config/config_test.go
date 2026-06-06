package config

import (
	"testing"
	"time"
)

func TestLoadFromLookupParsesEnvironment(t *testing.T) {
	values := map[string]string{
		"APP_ENV":                        "test",
		"HTTP_ADDR":                      ":9090",
		"CORS_ALLOWED_ORIGINS":           "http://localhost:3000,https://netquest.local",
		"REQUEST_TIMEOUT":                "7s",
		"POSTGRES_DSN":                   "postgres://user:pass@localhost:5432/netquest?sslmode=disable",
		"REDIS_ADDR":                     "redis:6379",
		"REDIS_DB":                       "2",
		"NATS_URL":                       "nats://nats:4222",
		"JWT_SECRET":                     "test-secret-with-at-least-thirty-two-characters",
		"JWT_ACCESS_TOKEN_TTL":           "30m",
		"PASSWORD_HASH_COST":             "11",
		"RATE_LIMIT_REQUESTS_PER_MINUTE": "42",
	}

	cfg, err := LoadFromLookup(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("LoadFromLookup returned error: %v", err)
	}

	if cfg.HTTP.Addr != ":9090" {
		t.Fatalf("unexpected HTTP addr: %s", cfg.HTTP.Addr)
	}
	if cfg.HTTP.RequestTimeout != 7*time.Second {
		t.Fatalf("unexpected request timeout: %s", cfg.HTTP.RequestTimeout)
	}
	if got := len(cfg.HTTP.CORSAllowedOrigins); got != 2 {
		t.Fatalf("expected 2 CORS origins, got %d", got)
	}
	if cfg.Redis.DB != 2 {
		t.Fatalf("unexpected redis db: %d", cfg.Redis.DB)
	}
	if cfg.RateLimit.RequestsPerMinute != 42 {
		t.Fatalf("unexpected rate limit: %d", cfg.RateLimit.RequestsPerMinute)
	}
}

func TestLoadFromLookupRejectsInvalidProductionSecret(t *testing.T) {
	values := map[string]string{
		"APP_ENV":            "production",
		"JWT_SECRET":         "too-short",
		"PASSWORD_HASH_COST": "12",
	}

	_, err := LoadFromLookup(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
