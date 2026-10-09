package config

// redisConfig — Redis для распределённого rate limiter'а и глубокого health-check.
type redisConfig struct {
	Addr     string // host:port
	Username string // ACL-пользователь Redis 6+; пусто — default
	Password string
	DB       int
	// TLSEnabled — подключаться по TLS (управляемый Redis в облаке обычно требует его).
	TLSEnabled bool
}

func loadRedisConfig(lookup LookupFunc) redisConfig {
	return redisConfig{
		Addr:       getString(lookup, "REDIS_ADDR", "localhost:6379"),
		Username:   getString(lookup, "REDIS_USERNAME", ""),
		Password:   getString(lookup, "REDIS_PASSWORD", ""),
		DB:         getInt(lookup, "REDIS_DB", 0),
		TLSEnabled: getBool(lookup, "REDIS_TLS_ENABLED", false),
	}
}
