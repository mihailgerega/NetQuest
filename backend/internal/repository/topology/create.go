package topology

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/converter"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// Create сохраняет новую версию топологии проекта с номером max(version)+1.
//
// Номер считается и вставляется в одной транзакции. От гонки двух параллельных
// сохранений она не спасает (оба прочитают один max), но вторую вставку отобьёт
// уникальный индекс (project_id, version) — сохранение упадёт, а не создаст
// две версии с одним номером.
//
// pgx.BeginFunc: BEGIN → fn(tx) → COMMIT, а если fn вернула ошибку — ROLLBACK.
func (r *repository) Create(ctx context.Context, topology model.Topology) (model.Topology, error) {
	var created model.Topology

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		const nextVersionQuery = `
			SELECT COALESCE(MAX(version), 0) + 1
			FROM topologies
			WHERE project_id = $1 AND deleted_at IS NULL`

		if err := tx.QueryRow(ctx, nextVersionQuery, topology.ProjectID).Scan(&topology.Version); err != nil {
			return fmt.Errorf("посчитать номер версии: %w", err)
		}

		// RETURNING без префикса t.: в INSERT таблица не соединяется с projects.
		const insertQuery = `
			INSERT INTO topologies (id, project_id, version, name, data, created_at, updated_at, created_by)
			VALUES ($1, $2, $3, $4, $5::jsonb, now(), now(), $6)
			RETURNING id::text AS id, project_id::text AS project_id, version, name, data::text AS data,
			          created_at, updated_at, deleted_at, created_by::text AS created_by`

		rows, err := tx.Query(ctx, insertQuery,
			topology.ID, topology.ProjectID, topology.Version, topology.Name, string(topology.Data), topology.CreatedBy)
		if err != nil {
			return fmt.Errorf("вставить версию топологии: %w", err)
		}

		rec, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[record.Topology])
		if err != nil {
			return fmt.Errorf("прочитать версию топологии: %w", err)
		}

		created = converter.TopologyToModel(rec)

		return nil
	})
	if err != nil {
		return model.Topology{}, fmt.Errorf("создать версию топологии: %w", err)
	}

	return created, nil
}
