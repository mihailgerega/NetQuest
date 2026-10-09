package config

// rateLimitConfig — ограничение частоты запросов с одного IP.
//
// Счётчик живёт в Redis под ключом <RedisPrefix>:<ip>:<номер минуты>, поэтому
// лимит общий для всех экземпляров API. Если Redis недоступен, лимитер
// переходит на счётчик в памяти процесса (см. internal/ratelimit).
type rateLimitConfig struct {
	// RequestsPerMinute — сколько запросов в минуту разрешено одному IP; 0 — без лимита.
	RequestsPerMinute int
	// Burst читается, но лимитером с фиксированным окном не используется:
	// оставлен, чтобы переход на token bucket не менял переменные окружения.
	Burst       int
	RedisPrefix string
}

func loadRateLimitConfig(lookup LookupFunc) rateLimitConfig {
	return rateLimitConfig{
		RequestsPerMinute: getInt(lookup, "RATE_LIMIT_REQUESTS_PER_MINUTE", 300),
		Burst:             getInt(lookup, "RATE_LIMIT_BURST", 30),
		RedisPrefix:       getString(lookup, "RATE_LIMIT_REDIS_PREFIX", "netquest:rl"),
	}
}
