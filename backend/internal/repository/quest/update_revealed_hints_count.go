package quest

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// UpdateRevealedHintsCount сохраняет число открытых подсказок. GREATEST не даёт
// счётчику уменьшиться, даже если два запроса пришли не по порядку.
func (r *repository) UpdateRevealedHintsCount(ctx context.Context, attemptID, userID string, count int) (model.Attempt, error) {
	const query = `
		UPDATE quest_attempts
		SET revealed_hints_count = GREATEST(revealed_hints_count, $3)
		WHERE id = $1 AND user_id = $2
		RETURNING ` + attemptColumns

	return r.queryAttempt(ctx, query, attemptID, userID, count)
}
