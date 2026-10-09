package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netquest/netquest/backend/internal/config"
)

// lookupFrom превращает map в источник переменных: тесты не трогают окружение
// процесса и могут идти параллельно.
func lookupFrom(values map[string]string) config.LookupFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

// TestLoadFromLookup: значения из окружения разбираются, списки чистятся,
// кривые значения и пустые строки заменяются значениями по умолчанию.
func TestLoadFromLookup(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFromLookup(lookupFrom(map[string]string{
		"APP_ENV":                        "test",
		"HTTP_ADDR":                      ":9090",
		"CORS_ALLOWED_ORIGINS":           "http://localhost:3000, ,https://netquest.local",
		"REQUEST_TIMEOUT":                "7s",
		"POSTGRES_DSN":                   "postgres://user:pass@localhost:5432/netquest?sslmode=disable",
		"REDIS_ADDR":                     "redis:6379",
		"REDIS_DB":                       "2",
		"NATS_URL":                       "nats://nats:4222",
		"JWT_SECRET":                     "test-secret-with-at-least-thirty-two-characters",
		"JWT_ACCESS_TOKEN_TTL":           "30m",
		"PASSWORD_HASH_COST":             "11",
		"RATE_LIMIT_REQUESTS_PER_MINUTE": "42",
		"HTTP_READ_TIMEOUT":              "десять секунд", // не разбирается → значение по умолчанию
		"SERVICE_NAME":                   "   ",           // пустое → значение по умолчанию
	}))
	require.NoError(t, err)

	assert.Equal(t, ":9090", cfg.HTTP.Addr)
	assert.Equal(t, 7*time.Second, cfg.HTTP.RequestTimeout)
	assert.Equal(t, []string{"http://localhost:3000", "https://netquest.local"}, cfg.HTTP.CORSAllowedOrigins)
	assert.Equal(t, 2, cfg.Redis.DB)
	assert.Equal(t, 30*time.Minute, cfg.Security.AccessTokenTTL)
	assert.Equal(t, 11, cfg.Security.PasswordHashCost)
	assert.Equal(t, 42, cfg.RateLimit.RequestsPerMinute)
	assert.Equal(t, 5*time.Second, cfg.HTTP.ReadTimeout)
	assert.Equal(t, "netquest-api", cfg.App.ServiceName)
}

// TestAccessTokenTTLAlias: прежнее имя ACCESS_TOKEN_TTL работает,
// а JWT_ACCESS_TOKEN_TTL главнее него.
func TestAccessTokenTTLAlias(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFromLookup(lookupFrom(map[string]string{"ACCESS_TOKEN_TTL": "20m"}))
	require.NoError(t, err)
	assert.Equal(t, 20*time.Minute, cfg.Security.AccessTokenTTL)

	cfg, err = config.LoadFromLookup(lookupFrom(map[string]string{"ACCESS_TOKEN_TTL": "20m", "JWT_ACCESS_TOKEN_TTL": "5m"}))
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, cfg.Security.AccessTokenTTL)
}

// TestValidate: каждое правило Validate называет переменную, которую надо поправить.
func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{name: "короткий секрет вне local", env: map[string]string{"APP_ENV": "production", "JWT_SECRET": "too-short"}, wantErr: "JWT_SECRET"},
		{name: "слабый bcrypt", env: map[string]string{"PASSWORD_HASH_COST": "4"}, wantErr: "PASSWORD_HASH_COST"},
		{name: "нулевой TTL access-токена", env: map[string]string{"JWT_ACCESS_TOKEN_TTL": "0s"}, wantErr: "JWT_ACCESS_TOKEN_TTL"},
		{name: "отрицательный лимит", env: map[string]string{"RATE_LIMIT_REQUESTS_PER_MINUTE": "-1"}, wantErr: "RATE_LIMIT_REQUESTS_PER_MINUTE"},
		{name: "нулевой лимит тела", env: map[string]string{"MAX_REQUEST_BODY_BYTES": "0"}, wantErr: "MAX_REQUEST_BODY_BYTES"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := config.LoadFromLookup(lookupFrom(tc.env))

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// TestDefaultsAreValid: без переменных окружения конфиг локальной разработки валиден.
func TestDefaultsAreValid(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFromLookup(lookupFrom(nil))
	require.NoError(t, err)

	assert.Equal(t, "local", cfg.App.Env)
	assert.Equal(t, ":8080", cfg.HTTP.Addr)
	assert.Equal(t, 15*time.Second, cfg.HTTP.ShutdownTimeout)
	assert.Equal(t, int32(10), cfg.Postgres.MaxConns)
	assert.True(t, cfg.Security.DemoAuthEnabled)
	assert.Equal(t, "migrations", cfg.Migrations.Dir)
}
