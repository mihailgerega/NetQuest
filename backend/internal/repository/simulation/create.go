package simulation

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/converter"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// Create сохраняет новую симуляцию и возвращает строку, как её записала база.
// Время запуска и завершения пока пустые — их проставит UpdateStatus.
func (r *repository) Create(ctx context.Context, simulation model.Simulation) (model.Simulation, error) {
	const query = `
		INSERT INTO simulations (id, project_id, topology_id, user_id, status, scenario, seed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, now(), now())
		RETURNING ` + simulationColumns

	rows, err := r.pool.Query(ctx, query,
		simulation.ID, simulation.ProjectID, simulation.TopologyID, simulation.UserID,
		simulation.Status, string(simulation.Scenario), simulation.Seed)
	if err != nil {
		return model.Simulation{}, fmt.Errorf("создать симуляцию: %w", err)
	}

	rec, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[record.Simulation])
	if err != nil {
		return model.Simulation{}, fmt.Errorf("прочитать созданную симуляцию: %w", err)
	}

	return converter.SimulationToModel(rec), nil
}
