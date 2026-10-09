package quest

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// GetAttemptForOwner читает попытку пользователя.
// Чужая или несуществующая → errs.ErrAttemptNotFound.
func (r *repository) GetAttemptForOwner(ctx context.Context, attemptID, userID string) (model.Attempt, error) {
	const query = `SELECT ` + attemptColumns + ` FROM quest_attempts WHERE id = $1 AND user_id = $2`

	return r.queryAttempt(ctx, query, attemptID, userID)
}
