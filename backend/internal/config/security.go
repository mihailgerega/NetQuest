package config

import "time"

// Значения по умолчанию для аутентификации.
const (
	// defaultJWTSecret — секрет для локальной разработки. Он длиннее 32 символов
	// и поэтому проходит Validate в любом окружении: вне local его обязательно
	// переопределяют через JWT_SECRET, иначе access-токены подписаны публично
	// известным ключом из репозитория.
	defaultJWTSecret         = "local-dev-secret-change-before-production-32"
	defaultAccessTokenTTL    = 15 * time.Minute
	defaultRefreshTokenTTL   = 30 * 24 * time.Hour
	defaultPasswordHashCost  = 12
	defaultHealthDeepTimeout = 3 * time.Second
)

// securityConfig — аутентификация: JWT, refresh-токены, хеширование паролей.
type securityConfig struct {
	// JWTIssuer — значение claim iss; токен с другим iss не принимается.
	JWTIssuer string
	// JWTSecret — ключ HMAC-SHA256 для подписи access-токенов.
	JWTSecret string
	// AccessTokenTTL — срок жизни access-токена (JWT). Короткий: отозвать JWT
	// до истечения нельзя, поэтому срок и есть окно риска при утечке.
	AccessTokenTTL time.Duration
	// RefreshTokenTTL — срок жизни refresh-токена в базе.
	RefreshTokenTTL time.Duration
	// PasswordHashCost — сложность bcrypt: 2^cost раундов.
	PasswordHashCost int
	// SecureCookie — ставить refresh-cookie с флагом Secure (только HTTPS).
	SecureCookie bool
	// DemoAuthEnabled — разрешён ли вход без пароля в общий demo-аккаунт.
	DemoAuthEnabled bool
	// HealthDeepTimeout — срок глубокой проверки зависимостей (/health/deep).
	HealthDeepTimeout time.Duration
}

// loadSecurityConfig читает настройки аутентификации.
//
// TTL access-токена ищется по двум именам: JWT_ACCESS_TOKEN_TTL главнее,
// ACCESS_TOKEN_TTL — прежнее имя переменной, оставлено для старых .env.
func loadSecurityConfig(lookup LookupFunc) securityConfig {
	accessTokenTTL := getDuration(lookup, "ACCESS_TOKEN_TTL", defaultAccessTokenTTL)

	return securityConfig{
		JWTIssuer:         getString(lookup, "JWT_ISSUER", "netquest"),
		JWTSecret:         getString(lookup, "JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL:    getDuration(lookup, "JWT_ACCESS_TOKEN_TTL", accessTokenTTL),
		RefreshTokenTTL:   getDuration(lookup, "REFRESH_TOKEN_TTL", defaultRefreshTokenTTL),
		PasswordHashCost:  getInt(lookup, "PASSWORD_HASH_COST", defaultPasswordHashCost),
		SecureCookie:      getBool(lookup, "SECURE_COOKIE", false),
		DemoAuthEnabled:   getBool(lookup, "DEMO_AUTH_ENABLED", true),
		HealthDeepTimeout: getDuration(lookup, "HEALTH_DEEP_TIMEOUT", defaultHealthDeepTimeout),
	}
}
