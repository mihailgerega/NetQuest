package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// Refresh обменивает действующий refresh-токен на новую пару токенов.
//
// Ротация: старый токен отзывается до выдачи нового. Украденный и уже
// использованный токен второй раз не сработает.
//
// Шаги не объединены в транзакцию: два параллельных Refresh одним токеном
// могут оба пройти поиск до отзыва и оба получить новые токены.
func (s *service) Refresh(ctx context.Context, refreshToken string, client input.ClientInfo) (model.AuthTokens, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return model.AuthTokens{}, errs.ErrRefreshTokenRequired
	}

	now := time.Now().UTC()

	// Ошибки репозитория отдаём без обёртки: ErrRefreshTokenInvalid (401)
	// и ErrUserNotFound (404) — ответы клиенту, а сбои базы и так станут 500.
	stored, err := s.refreshTokenRepository.FindActiveByHash(ctx, hashToken(refreshToken), now)
	if err != nil {
		return model.AuthTokens{}, err
	}

	if err := s.refreshTokenRepository.RevokeByHash(ctx, stored.TokenHash, now); err != nil {
		return model.AuthTokens{}, fmt.Errorf("отозвать старый refresh-токен: %w", err)
	}

	user, err := s.userRepository.FindByID(ctx, stored.UserID)
	if err != nil {
		return model.AuthTokens{}, err
	}

	return s.issueTokens(ctx, user, client)
}
