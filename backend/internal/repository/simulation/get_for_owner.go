package simulation

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/converter"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// GetForOwner читает симуляцию, если её проект принадлежит ownerID.
// Чужая, удалённая или несуществующая → errs.ErrSimulationNotFound.
func (r *repository) GetForOwner(ctx context.Context, simulationID, ownerID string) (model.Simulation, error) {
	const query = `
		SELECT s.id::text AS id, s.project_id::text AS project_id, s.topology_id::text AS topology_id,
		       s.user_id::text AS user_id, s.status, s.scenario::text AS scenario, s.seed,
		       s.started_at, s.finished_at, s.error_message, s.created_at, s.updated_at
		FROM simulations s
		INNER JOIN projects p ON p.id = s.project_id
		WHERE s.id = $1 AND p.owner_id = $2 AND s.deleted_at IS NULL AND p.deleted_at IS NULL`

	rows, err := r.pool.Query(ctx, query, simulationID, ownerID)
	if err != nil {
		return model.Simulation{}, fmt.Errorf("получить симуляцию: %w", err)
	}

	rec, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[record.Simulation])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Simulation{}, errs.ErrSimulationNotFound
		}

		return model.Simulation{}, fmt.Errorf("прочитать симуляцию: %w", err)
	}

	return converter.SimulationToModel(rec), nil
}
