package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// refreshTokenBytes — длина refresh-токена: 256 случайных бит, перебором не угадать.
const refreshTokenBytes = 32

// issueTokens выдаёт пользователю access-токен (JWT) и refresh-токен и
// сохраняет хеш refresh-токена вместе с данными клиента.
func (s *service) issueTokens(ctx context.Context, user model.User, client input.ClientInfo) (model.AuthTokens, error) {
	now := time.Now().UTC()

	accessToken, err := s.tokenIssuer.GenerateAccessToken(user.ID, user.Email, user.Role, now)
	if err != nil {
		return model.AuthTokens{}, fmt.Errorf("выпустить access-токен: %w", err)
	}

	refreshToken, err := newOpaqueToken()
	if err != nil {
		return model.AuthTokens{}, err
	}

	if err := s.refreshTokenRepository.Create(ctx, model.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: hashToken(refreshToken),
		UserAgent: client.UserAgent,
		IPAddress: client.IPAddress,
		ExpiresAt: now.Add(s.settings.RefreshTokenTTL),
	}); err != nil {
		return model.AuthTokens{}, fmt.Errorf("сохранить refresh-токен: %w", err)
	}

	return model.AuthTokens{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    model.TokenTypeBearer,
		ExpiresIn:    int64(s.settings.AccessTokenTTL.Seconds()),
	}, nil
}

// newOpaqueToken генерирует refresh-токен: случайные байты из crypto/rand
// в base64url без паддинга — безопасно для cookie и URL.
func newOpaqueToken() (string, error) {
	buf := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("сгенерировать refresh-токен: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// hashToken — SHA-256 токена в hex. Соль не нужна: токен и так случайный
// и длинный, словарной атаки на него нет (в отличие от пароля).
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
