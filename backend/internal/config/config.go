// Package config — конфигурация NetQuest API и CLI миграций.
//
// Источник значений — только переменные окружения: в Docker и на сервере их
// задаёт docker compose из .env, локально — shell или .env.local. Для каждой
// переменной есть значение по умолчанию для локального запуска.
//
// Правила чтения (env.go): пустая строка или строка из пробелов — то же, что
// «не задано»; значение, которое не разбирается (DB=abc, TTL=10 минут), тоже
// молча заменяется значением по умолчанию. Проверка осмысленности значений —
// в Validate, она уже падает с ошибкой.
//
// Каждая секция описана в своём файле: app.go, http.go, postgres.go, redis.go,
// nats.go, security.go, rate_limit.go, migrations.go.
package config

import (
	"errors"
	"os"
	"strings"
)

const (
	// envLocal — окружение локальной разработки: только в нём разрешён
	// короткий JWT_SECRET по умолчанию.
	envLocal = "local"
	// minJWTSecretLength — минимум символов секрета вне local: HS256-ключ
	// короче 32 байт перебирается заметно проще.
	minJWTSecretLength = 32
	// minPasswordHashCost — минимальная сложность bcrypt: 2^10 раундов.
	minPasswordHashCost = 10
)

// LookupFunc — источник переменных окружения. В боевом коде — os.LookupEnv,
// в тестах — map: так тесты не трогают окружение процесса и идут параллельно.
type LookupFunc func(string) (string, bool)

// Config — вся конфигурация сервиса.
type Config struct {
	App        appConfig
	HTTP       httpConfig
	Postgres   postgresConfig
	Redis      redisConfig
	NATS       natsConfig
	Security   securityConfig
	RateLimit  rateLimitConfig
	Migrations migrationsConfig
}

// Load читает конфигурацию из переменных окружения процесса и проверяет её.
func Load() (*Config, error) {
	return LoadFromLookup(os.LookupEnv)
}

// LoadFromLookup читает конфигурацию из произвольного источника и проверяет её.
func LoadFromLookup(lookup LookupFunc) (*Config, error) {
	cfg := &Config{
		App:        loadAppConfig(lookup),
		HTTP:       loadHTTPConfig(lookup),
		Postgres:   loadPostgresConfig(lookup),
		Redis:      loadRedisConfig(lookup),
		NATS:       loadNATSConfig(lookup),
		Security:   loadSecurityConfig(lookup),
		RateLimit:  loadRateLimitConfig(lookup),
		Migrations: loadMigrationsConfig(lookup),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate проверяет, что с такой конфигурацией сервис может работать.
// Возвращает первую найденную проблему: имя переменной в тексте сразу
// говорит, что поправить.
func (c *Config) Validate() error {
	checks := []struct {
		failed  bool
		message string
	}{
		{strings.TrimSpace(c.HTTP.Addr) == "", "HTTP_ADDR не должен быть пустым"},
		{strings.TrimSpace(c.Postgres.DSN) == "", "POSTGRES_DSN не должен быть пустым"},
		{strings.TrimSpace(c.Redis.Addr) == "", "REDIS_ADDR не должен быть пустым"},
		{strings.TrimSpace(c.NATS.URL) == "", "NATS_URL не должен быть пустым"},
		{c.HTTP.MaxRequestBodyBytes <= 0, "MAX_REQUEST_BODY_BYTES должен быть больше нуля"},
		{c.HTTP.JSONBodyLimitBytes <= 0, "JSON_BODY_LIMIT_BYTES должен быть больше нуля"},
		{c.Security.AccessTokenTTL <= 0, "JWT_ACCESS_TOKEN_TTL должен быть больше нуля"},
		{c.Security.RefreshTokenTTL <= 0, "REFRESH_TOKEN_TTL должен быть больше нуля"},
		{c.Security.PasswordHashCost < minPasswordHashCost, "PASSWORD_HASH_COST должен быть не меньше 10"},
		{c.RateLimit.RequestsPerMinute < 0, "RATE_LIMIT_REQUESTS_PER_MINUTE не может быть отрицательным"},
		{c.App.Env != envLocal && len(c.Security.JWTSecret) < minJWTSecretLength, "JWT_SECRET вне local должен быть не короче 32 символов"},
	}

	for _, check := range checks {
		if check.failed {
			return errors.New(check.message)
		}
	}

	return nil
}
