// Package ratelimit — ограничение частоты запросов с одного IP:
// фиксированное окно в одну минуту.
//
// Основной счётчик — в Redis (INCR по ключу <префикс>:<ip>:<номер минуты>),
// поэтому лимит общий для всех экземпляров API. Если Redis недоступен,
// лимитер не отключается и не роняет запросы, а считает в памяти процесса
// (LocalLimiter) — лимит тогда действует на каждый экземпляр отдельно.
package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/httpx"
)

const (
	// windowSeconds — длина окна: лимит считается на минуту.
	windowSeconds = 60
	// keyTTL — сколько хранится счётчик окна. С запасом в одно окно: ключ
	// прошлой минуты ещё может понадобиться запросу, пришедшему на стыке.
	keyTTL = 2 * time.Minute
	// retryAfterSeconds — через сколько секунд клиенту стоит повторить (заголовок Retry-After).
	retryAfterSeconds = "60"
)

// Limiter — middleware ограничения частоты. Безопасен для параллельных запросов:
// redis.Client и LocalLimiter потокобезопасны.
type Limiter struct {
	redis  *redis.Client // nil — только счётчик в памяти
	local  *LocalLimiter
	limit  int // запросов в минуту на IP; 0 и меньше — без ограничения
	prefix string
}

// New создаёт лимитер на requestsPerMinute запросов в минуту с одного IP.
func New(redisClient *redis.Client, requestsPerMinute int, prefix string) *Limiter {
	return &Limiter{
		redis:  redisClient,
		local:  NewLocalLimiter(requestsPerMinute),
		limit:  requestsPerMinute,
		prefix: prefix,
	}
}

// Middleware отклоняет запрос сверх лимита ответом 429 с Retry-After.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.limit <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		allowed, err := l.allow(r.Context(), httpx.ClientIP(r))
		if err != nil {
			slog.WarnContext(r.Context(), "rate limiter перешёл на счётчик в памяти", "error", err)
		}

		if !allowed {
			w.Header().Set("Retry-After", retryAfterSeconds)
			httpx.WriteError(w, r, &errs.RateLimitError{Limit: l.limit})

			return
		}

		next.ServeHTTP(w, r)
	})
}

// allow засчитывает запрос и сообщает, укладывается ли он в лимит.
// Ошибка Redis не отклоняет запрос: решение принимает счётчик в памяти,
// а ошибка возвращается только для лога.
func (l *Limiter) allow(ctx context.Context, key string) (bool, error) {
	if l.redis == nil {
		return l.local.Allow(key), nil
	}

	window := time.Now().UTC().Unix() / windowSeconds
	redisKey := fmt.Sprintf("%s:%s:%d", l.prefix, key, window)

	count, err := l.redis.Incr(ctx, redisKey).Result()
	if err != nil {
		return l.local.Allow(key), err
	}

	// TTL ставит только первый запрос окна. Если EXPIRE не дошёл, ключ останется
	// без срока — но он уникален для минуты, и в следующем окне не используется.
	if count == 1 {
		_ = l.redis.Expire(ctx, redisKey, keyTTL).Err() //nolint:gosec // G104: см. комментарий выше
	}

	return count <= int64(l.limit), nil
}
