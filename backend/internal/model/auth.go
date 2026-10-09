package model

import "time"

// TokenTypeBearer — тип access-токена в ответе: клиент шлёт его как
// Authorization: Bearer <accessToken>.
const TokenTypeBearer = "Bearer"

// Principal — кто делает запрос: данные из проверенного access-токена (JWT).
// Middleware кладёт его в ctx запроса, хендлеры достают через auth.PrincipalFromContext.
type Principal struct {
	UserID string
	Email  string
	Role   string
}

// RefreshToken — долгоживущий токен для обновления access-токена.
//
// В базе лежит не сам токен, а его SHA-256 (TokenHash): утечка таблицы
// не даёт войти. Токен одноразовый — при Refresh старый отзывается (RevokedAt)
// и выдаётся новый.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	UserAgent string
	IPAddress string
	ExpiresAt time.Time
	RevokedAt *time.Time // nil — токен не отозван
	CreatedAt time.Time
}

// AuthTokens — результат входа: пользователь и выданная пара токенов.
//
// RefreshToken здесь в открытом виде — сервис только что его сгенерировал.
// API кладёт его в httpOnly-cookie и в тело ответа не пишет.
type AuthTokens struct {
	User         User
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64 // срок жизни access-токена в секундах
}
