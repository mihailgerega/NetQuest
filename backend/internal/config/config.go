package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type LookupFunc func(string) (string, bool)

type Config struct {
	App        AppConfig
	HTTP       HTTPConfig
	Postgres   PostgresConfig
	Redis      RedisConfig
	NATS       NATSConfig
	Security   SecurityConfig
	RateLimit  RateLimitConfig
	Migrations MigrationsConfig
}

type AppConfig struct {
	Env         string
	ServiceName string
}

type HTTPConfig struct {
	Addr                string
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	RequestTimeout      time.Duration
	ShutdownTimeout     time.Duration
	MaxRequestBodyBytes int64
	JSONBodyLimitBytes  int64
	CORSAllowedOrigins  []string
}

type PostgresConfig struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Addr       string
	Username   string
	Password   string
	DB         int
	TLSEnabled bool
}

type NATSConfig struct {
	URL     string
	Timeout time.Duration
}

type SecurityConfig struct {
	JWTIssuer         string
	JWTSecret         string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	PasswordHashCost  int
	SecureCookie      bool
	DemoAuthEnabled   bool
	HealthDeepTimeout time.Duration
}

type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
	RedisPrefix       string
}

type MigrationsConfig struct {
	Dir string
}

func Load() (Config, error) {
	return LoadFromLookup(os.LookupEnv)
}

func LoadFromLookup(lookup LookupFunc) (Config, error) {
	cfg := Config{
		App: AppConfig{
			Env:         getString(lookup, "APP_ENV", "local"),
			ServiceName: getString(lookup, "SERVICE_NAME", "netquest-api"),
		},
		HTTP: HTTPConfig{
			Addr:                getString(lookup, "HTTP_ADDR", ":8080"),
			ReadTimeout:         getDuration(lookup, "HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout:        getDuration(lookup, "HTTP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:         getDuration(lookup, "HTTP_IDLE_TIMEOUT", 60*time.Second),
			RequestTimeout:      getDuration(lookup, "REQUEST_TIMEOUT", 5*time.Second),
			ShutdownTimeout:     getDuration(lookup, "SHUTDOWN_TIMEOUT", 15*time.Second),
			MaxRequestBodyBytes: getInt64(lookup, "MAX_REQUEST_BODY_BYTES", 1<<20),
			JSONBodyLimitBytes:  getInt64(lookup, "JSON_BODY_LIMIT_BYTES", 1<<20),
			CORSAllowedOrigins:  getCSV(lookup, "CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		},
		Postgres: PostgresConfig{
			DSN:             getString(lookup, "POSTGRES_DSN", "postgres://netquest:netquest@localhost:5432/netquest?sslmode=disable"),
			MaxConns:        int32(getInt(lookup, "POSTGRES_MAX_CONNS", 10)),
			MinConns:        int32(getInt(lookup, "POSTGRES_MIN_CONNS", 1)),
			ConnMaxLifetime: getDuration(lookup, "POSTGRES_CONN_MAX_LIFETIME", time.Hour),
		},
		Redis: RedisConfig{
			Addr:       getString(lookup, "REDIS_ADDR", "localhost:6379"),
			Username:   getString(lookup, "REDIS_USERNAME", ""),
			Password:   getString(lookup, "REDIS_PASSWORD", ""),
			DB:         getInt(lookup, "REDIS_DB", 0),
			TLSEnabled: getBool(lookup, "REDIS_TLS_ENABLED", false),
		},
		NATS: NATSConfig{
			URL:     getString(lookup, "NATS_URL", "nats://localhost:4222"),
			Timeout: getDuration(lookup, "NATS_TIMEOUT", 2*time.Second),
		},
		Security: SecurityConfig{
			JWTIssuer:         getString(lookup, "JWT_ISSUER", "netquest"),
			JWTSecret:         getString(lookup, "JWT_SECRET", "local-dev-secret-change-before-production-32"),
			AccessTokenTTL:    getDuration(lookup, "JWT_ACCESS_TOKEN_TTL", getDuration(lookup, "ACCESS_TOKEN_TTL", 15*time.Minute)),
			RefreshTokenTTL:   getDuration(lookup, "REFRESH_TOKEN_TTL", 30*24*time.Hour),
			PasswordHashCost:  getInt(lookup, "PASSWORD_HASH_COST", 12),
			SecureCookie:      getBool(lookup, "SECURE_COOKIE", false),
			DemoAuthEnabled:   getBool(lookup, "DEMO_AUTH_ENABLED", true),
			HealthDeepTimeout: getDuration(lookup, "HEALTH_DEEP_TIMEOUT", 3*time.Second),
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getInt(lookup, "RATE_LIMIT_REQUESTS_PER_MINUTE", 300),
			Burst:             getInt(lookup, "RATE_LIMIT_BURST", 30),
			RedisPrefix:       getString(lookup, "RATE_LIMIT_REDIS_PREFIX", "netquest:rl"),
		},
		Migrations: MigrationsConfig{
			Dir: getString(lookup, "MIGRATIONS_DIR", "migrations"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.HTTP.Addr) == "" {
		return fmt.Errorf("HTTP_ADDR must not be empty")
	}
	if strings.TrimSpace(c.Postgres.DSN) == "" {
		return fmt.Errorf("POSTGRES_DSN must not be empty")
	}
	if strings.TrimSpace(c.Redis.Addr) == "" {
		return fmt.Errorf("REDIS_ADDR must not be empty")
	}
	if strings.TrimSpace(c.NATS.URL) == "" {
		return fmt.Errorf("NATS_URL must not be empty")
	}
	if c.HTTP.MaxRequestBodyBytes <= 0 {
		return fmt.Errorf("MAX_REQUEST_BODY_BYTES must be positive")
	}
	if c.HTTP.JSONBodyLimitBytes <= 0 {
		return fmt.Errorf("JSON_BODY_LIMIT_BYTES must be positive")
	}
	if c.Security.AccessTokenTTL <= 0 {
		return fmt.Errorf("JWT_ACCESS_TOKEN_TTL must be positive")
	}
	if c.Security.RefreshTokenTTL <= 0 {
		return fmt.Errorf("REFRESH_TOKEN_TTL must be positive")
	}
	if c.Security.PasswordHashCost < 10 {
		return fmt.Errorf("PASSWORD_HASH_COST must be at least 10")
	}
	if c.RateLimit.RequestsPerMinute < 0 {
		return fmt.Errorf("RATE_LIMIT_REQUESTS_PER_MINUTE must not be negative")
	}
	if c.App.Env != "local" && len(c.Security.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters outside local")
	}

	return nil
}

func getString(lookup LookupFunc, key, fallback string) string {
	if value, ok := lookup(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func getCSV(lookup LookupFunc, key string, fallback []string) []string {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func getBool(lookup LookupFunc, key string, fallback bool) bool {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getInt(lookup LookupFunc, key string, fallback int) int {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getInt64(lookup LookupFunc, key string, fallback int64) int64 {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getDuration(lookup LookupFunc, key string, fallback time.Duration) time.Duration {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
