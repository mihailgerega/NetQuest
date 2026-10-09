package quest

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/netquest/netquest/backend/internal/model"
)

// UpdateAttemptAfterCheck записывает итог проверки в попытку: новый статус,
// +1 к числу проверок и сам результат.
//
// completedAt пишется, только если задан (квест пройден): COALESCE сохраняет
// время первого прохождения, и провал после успеха его не стирает.
func (r *repository) UpdateAttemptAfterCheck(
	ctx context.Context,
	attemptID, userID string,
	status model.AttemptStatus,
	result model.QuestResult,
	completedAt *time.Time,
) (model.Attempt, error) {
	const query = `
		UPDATE quest_attempts
		SET status = $3,
		    attempts_count = attempts_count + 1,
		    last_check_result = $4::jsonb,
		    completed_at = COALESCE($5::timestamptz, completed_at)
		WHERE id = $1 AND user_id = $2
		RETURNING ` + attemptColumns

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return model.Attempt{}, fmt.Errorf("сериализовать результат проверки: %w", err)
	}

	return r.queryAttempt(ctx, query, attemptID, userID, status, string(resultJSON), completedAt)
}
