package simulation

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/converter"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// ListEventsForOwner возвращает события симуляции по порядку, если её проект
// принадлежит ownerID. Чужая симуляция даёт пустой список, а не ошибку:
// существование проверяет GetForOwner.
func (r *repository) ListEventsForOwner(ctx context.Context, simulationID, ownerID string) ([]model.Event, error) {
	const query = `
		SELECT e.id::text AS id, e.simulation_id::text AS simulation_id, e.sequence_number, e.timestamp_ms,
		       e.type, e.severity,
		       COALESCE(e.packet_id, '') AS packet_id, COALESCE(e.source_node_id, '') AS source_node_id,
		       COALESCE(e.target_node_id, '') AS target_node_id,
		       e.message, e.details::text AS details
		FROM simulation_events e
		INNER JOIN simulations s ON s.id = e.simulation_id
		INNER JOIN projects p ON p.id = s.project_id
		WHERE e.simulation_id = $1 AND p.owner_id = $2 AND s.deleted_at IS NULL AND p.deleted_at IS NULL
		ORDER BY e.sequence_number ASC`

	rows, err := r.pool.Query(ctx, query, simulationID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("получить события симуляции: %w", err)
	}

	recs, err := pgx.CollectRows(rows, pgx.RowToStructByName[record.SimulationEvent])
	if err != nil {
		return nil, fmt.Errorf("прочитать события симуляции: %w", err)
	}

	return converter.SimulationEventsToModel(recs)
}
