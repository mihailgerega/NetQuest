package auth

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// Me возвращает профиль пользователя из access-токена. Ошибку не оборачиваем:
// удалённый после выдачи токена пользователь — ErrUserNotFound (404).
func (s *service) Me(ctx context.Context, userID string) (model.User, error) {
	return s.userRepository.FindByID(ctx, userID)
}
