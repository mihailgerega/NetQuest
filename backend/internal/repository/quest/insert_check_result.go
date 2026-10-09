package quest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// InsertCheckResult сохраняет результат проверки в историю quest_check_results.
// passed и score дублируются колонками, чтобы их можно было агрегировать
// SQL-запросом, не разбирая jsonb.
func (r *repository) InsertCheckResult(ctx context.Context, id, attemptID, userID string, result model.QuestResult) error {
	const query = `
		INSERT INTO quest_check_results (id, attempt_id, user_id, passed, score, result)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)`

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("сериализовать результат проверки для истории: %w", err)
	}

	if _, err := r.pool.Exec(ctx, query, id, attemptID, userID, result.Passed, result.Score, string(resultJSON)); err != nil {
		return fmt.Errorf("сохранить результат проверки: %w", err)
	}

	return nil
}
