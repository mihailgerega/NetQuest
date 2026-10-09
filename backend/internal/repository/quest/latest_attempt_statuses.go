package quest

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// LatestAttemptStatuses возвращает статус последней (по updated_at) попытки
// пользователя по каждому квесту: quest_id → статус. Квестов без попыток
// в результате нет.
//
// DISTINCT ON (quest_id) с ORDER BY quest_id, updated_at DESC берёт для каждого
// квеста первую строку — самую свежую попытку — одним проходом по индексу.
func (r *repository) LatestAttemptStatuses(ctx context.Context, userID string) (map[string]model.AttemptStatus, error) {
	const query = `
		SELECT DISTINCT ON (quest_id) quest_id, status
		FROM quest_attempts
		WHERE user_id = $1
		ORDER BY quest_id, updated_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("получить статусы попыток: %w", err)
	}

	recs, err := pgx.CollectRows(rows, pgx.RowToStructByName[record.AttemptStatus])
	if err != nil {
		return nil, fmt.Errorf("прочитать статусы попыток: %w", err)
	}

	statuses := make(map[string]model.AttemptStatus, len(recs))
	for _, rec := range recs {
		statuses[rec.QuestID] = model.AttemptStatus(rec.Status)
	}

	return statuses, nil
}
