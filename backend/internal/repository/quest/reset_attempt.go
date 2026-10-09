package quest

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// ResetAttempt возвращает попытку в начальное состояние: in_progress,
// без проверок, открытых подсказок, результата и времени прохождения.
func (r *repository) ResetAttempt(ctx context.Context, attemptID, userID string) (model.Attempt, error) {
	const query = `
		UPDATE quest_attempts
		SET status = 'in_progress',
		    attempts_count = 0,
		    revealed_hints_count = 0,
		    last_check_result = NULL,
		    completed_at = NULL
		WHERE id = $1 AND user_id = $2
		RETURNING ` + attemptColumns

	return r.queryAttempt(ctx, query, attemptID, userID)
}
