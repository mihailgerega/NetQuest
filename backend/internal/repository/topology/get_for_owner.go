package topology

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

// GetForOwner читает версию топологии, если её проект принадлежит ownerID.
// Чужая, удалённая или несуществующая → errs.ErrTopologyNotFound.
func (r *repository) GetForOwner(ctx context.Context, topologyID, ownerID string) (model.Topology, error) {
	const query = `
		SELECT ` + topologyColumns + `
		FROM topologies t
		INNER JOIN projects p ON p.id = t.project_id
		WHERE t.id = $1
		  AND p.owner_id = $2
		  AND t.deleted_at IS NULL
		  AND p.deleted_at IS NULL`

	rows, err := r.pool.Query(ctx, query, topologyID, ownerID)
	if err != nil {
		return model.Topology{}, fmt.Errorf("получить версию топологии: %w", err)
	}

	rec, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[record.Topology])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Topology{}, errs.ErrTopologyNotFound
		}

		return model.Topology{}, fmt.Errorf("прочитать версию топологии: %w", err)
	}

	return converter.TopologyToModel(rec), nil
}
