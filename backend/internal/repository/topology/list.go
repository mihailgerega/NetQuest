package topology

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/converter"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// ListForProjectOwner возвращает версии топологии проекта владельца:
// сначала самые новые.
func (r *repository) ListForProjectOwner(ctx context.Context, projectID, ownerID string) ([]model.Topology, error) {
	const query = `
		SELECT ` + topologyColumns + `
		FROM topologies t
		INNER JOIN projects p ON p.id = t.project_id
		WHERE t.project_id = $1
		  AND p.owner_id = $2
		  AND t.deleted_at IS NULL
		  AND p.deleted_at IS NULL
		ORDER BY t.version DESC, t.created_at DESC`

	rows, err := r.pool.Query(ctx, query, projectID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("получить версии топологии: %w", err)
	}

	recs, err := pgx.CollectRows(rows, pgx.RowToStructByName[record.Topology])
	if err != nil {
		return nil, fmt.Errorf("прочитать версии топологии: %w", err)
	}

	return converter.TopologiesToModel(recs), nil
}
