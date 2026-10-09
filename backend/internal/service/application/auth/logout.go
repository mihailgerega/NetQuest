package auth

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Logout отзывает refresh-токен. Без токена делать нечего — это не ошибка:
// logout должен «получаться» всегда, даже если cookie уже нет.
//
// Access-токен остаётся действительным до истечения: JWT не отзывается,
// поэтому его срок и сделан коротким.
func (s *service) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil
	}

	if err := s.refreshTokenRepository.RevokeByHash(ctx, hashToken(refreshToken), time.Now().UTC()); err != nil {
		return fmt.Errorf("отозвать refresh-токен: %w", err)
	}

	return nil
}
