package simulation

import (
	"context"
	"fmt"
	"time"

	"github.com/netquest/netquest/backend/internal/model"
)

// UpdateStatus меняет статус симуляции.
//
// errorMessage записывается всегда (nil → NULL), а startedAt и finishedAt —
// только если заданы: COALESCE оставляет прежнее значение для nil. Так один
// метод и отмечает старт (running + startedAt), и завершение (статус + finishedAt).
func (r *repository) UpdateStatus(
	ctx context.Context,
	id string,
	status model.SimulationStatus,
	errorMessage *string,
	startedAt, finishedAt *time.Time,
) error {
	const query = `
		UPDATE simulations
		SET status = $2,
		    error_message = $3,
		    started_at = COALESCE($4, started_at),
		    finished_at = COALESCE($5, finished_at),
		    updated_at = now()
		WHERE id = $1`

	if _, err := r.pool.Exec(ctx, query, id, status, errorMessage, startedAt, finishedAt); err != nil {
		return fmt.Errorf("обновить статус симуляции: %w", err)
	}

	return nil
}
