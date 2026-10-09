package quest

import (
	"context"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// CreateAttempt сохраняет новую попытку и возвращает строку, как её записала база.
func (r *repository) CreateAttempt(ctx context.Context, attempt model.Attempt) (model.Attempt, error) {
	const query = `
		INSERT INTO quest_attempts (id, quest_id, user_id, project_id, current_topology_id, status, attempts_count, revealed_hints_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING ` + attemptColumns

	created, err := r.queryAttempt(ctx, query,
		attempt.ID, attempt.QuestID, attempt.UserID, attempt.ProjectID, attempt.CurrentTopologyID,
		attempt.Status, attempt.AttemptsCount, attempt.RevealedHintsCount)
	if err != nil {
		return model.Attempt{}, fmt.Errorf("создать попытку: %w", err)
	}

	return created, nil
}
