package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/netquest/netquest/backend/internal/model"
)

// AccessClaims — содержимое access-токена: стандартные claims (sub — ID
// пользователя, iss, iat, exp) плюс email и роль, чтобы middleware не ходил
// за ними в базу на каждый запрос.
//
// Формат claims — контракт: токены, выпущенные до обновления сервиса, должны
// продолжать проверяться, а фронтенд может читать их payload.
type AccessClaims struct {
	Email string `json:"email,omitempty"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager выпускает и проверяет access-токены. Неизменяем после создания —
// безопасен для параллельных запросов.
type JWTManager struct {
	issuer string
	secret []byte
	ttl    time.Duration
}

// NewJWTManager создаёт менеджер токенов с ключом HMAC и сроком жизни токена.
func NewJWTManager(issuer, secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{
		issuer: issuer,
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// GenerateAccessToken выпускает токен пользователя, действующий ttl от now.
// now передаётся явно: сервис берёт одно время на всю выдачу токенов.
func (m *JWTManager) GenerateAccessToken(userID, email, role string, now time.Time) (string, error) {
	claims := AccessClaims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("подписать access-токен: %w", err)
	}

	return signed, nil
}

// ParseAccessToken проверяет токен и возвращает, кому он выдан.
//
// Проверяются подпись, срок (exp), издатель (iss) и наличие sub. Алгоритм
// подписи сверяется явно: без этого токен с "alg": "none" или подписанный
// другим алгоритмом мог бы пройти проверку (классическая атака на JWT).
func (m *JWTManager) ParseAccessToken(raw string) (model.Principal, error) {
	claims := &AccessClaims{}

	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("неожиданный алгоритм подписи %s", token.Method.Alg())
		}

		return m.secret, nil
	}, jwt.WithIssuer(m.issuer))
	if err != nil {
		return model.Principal{}, fmt.Errorf("разобрать access-токен: %w", err)
	}

	if !token.Valid {
		return model.Principal{}, errors.New("access-токен недействителен")
	}

	if claims.Subject == "" {
		return model.Principal{}, errors.New("в access-токене нет sub")
	}

	return model.Principal{
		UserID: claims.Subject,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}
