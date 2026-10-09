package quest

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// GetAttempt возвращает попытку пользователя; чужая → errs.ErrAttemptNotFound.
func (s *service) GetAttempt(ctx context.Context, userID, attemptID string) (model.Attempt, error) {
	return s.questRepository.GetAttemptForOwner(ctx, attemptID, userID)
}
