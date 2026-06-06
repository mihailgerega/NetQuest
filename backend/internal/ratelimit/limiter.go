package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/netquest/netquest/backend/internal/httpx"
	"github.com/netquest/netquest/backend/pkg/apperrors"
	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	redis  *redis.Client
	local  *LocalLimiter
	limit  int
	prefix string
	logger *slog.Logger
}

func New(redisClient *redis.Client, requestsPerMinute int, prefix string, logger *slog.Logger) *Limiter {
	return &Limiter{
		redis:  redisClient,
		local:  NewLocalLimiter(requestsPerMinute),
		limit:  requestsPerMinute,
		prefix: prefix,
		logger: logger,
	}
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l == nil || l.limit <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		key := httpx.ClientIP(r)
		allowed, err := l.allow(r.Context(), key)
		if err != nil && l.logger != nil {
			l.logger.Warn("rate limiter degraded to local memory", slog.String("error", err.Error()))
		}
		if !allowed {
			w.Header().Set("Retry-After", "60")
			httpx.WriteError(w, r, apperrors.WithDetails(http.StatusTooManyRequests, "rate_limited", "rate limit exceeded", map[string]any{
				"limit":  l.limit,
				"window": "1m",
			}))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) allow(ctx context.Context, key string) (bool, error) {
	if l.redis == nil {
		return l.local.Allow(key), nil
	}

	window := time.Now().UTC().Unix() / 60
	redisKey := fmt.Sprintf("%s:%s:%d", l.prefix, key, window)
	count, err := l.redis.Incr(ctx, redisKey).Result()
	if err != nil {
		return l.local.Allow(key), err
	}
	if count == 1 {
		_ = l.redis.Expire(ctx, redisKey, 2*time.Minute).Err()
	}
	return count <= int64(l.limit), nil
}

type LocalLimiter struct {
	mu      sync.Mutex
	limit   int
	windows map[string]localWindow
}

type localWindow struct {
	Count    int
	ResetAt  time.Time
	LastSeen time.Time
}

func NewLocalLimiter(limit int) *LocalLimiter {
	return &LocalLimiter{
		limit:   limit,
		windows: make(map[string]localWindow),
	}
}

func (l *LocalLimiter) Allow(key string) bool {
	if l.limit <= 0 {
		return true
	}

	now := time.Now()
	resetAt := now.Truncate(time.Minute).Add(time.Minute)

	l.mu.Lock()
	defer l.mu.Unlock()

	for existingKey, window := range l.windows {
		if now.Sub(window.LastSeen) > 5*time.Minute {
			delete(l.windows, existingKey)
		}
	}

	window := l.windows[key]
	if now.After(window.ResetAt) {
		window = localWindow{ResetAt: resetAt}
	}
	window.Count++
	window.LastSeen = now
	l.windows[key] = window

	return window.Count <= l.limit
}
