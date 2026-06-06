package topology

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/netquest/netquest/backend/pkg/apperrors"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListForProjectOwner(ctx context.Context, projectID, ownerID string) ([]Topology, error) {
	if r.db == nil {
		return nil, apperrors.Internal("postgres is not configured")
	}

	rows, err := r.db.Query(ctx, `
		SELECT t.id::text, t.project_id::text, t.version, t.name, t.data::text,
		       t.created_at, t.updated_at, t.deleted_at, t.created_by::text
		FROM topologies t
		INNER JOIN projects p ON p.id = t.project_id
		WHERE t.project_id = $1
		  AND p.owner_id = $2
		  AND t.deleted_at IS NULL
		  AND p.deleted_at IS NULL
		ORDER BY t.version DESC, t.created_at DESC
	`, projectID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list topologies: %w", err)
	}
	defer rows.Close()

	items := make([]Topology, 0)
	for rows.Next() {
		var item Topology
		var dataText string
		if err := rows.Scan(
			&item.ID,
			&item.ProjectID,
			&item.Version,
			&item.Name,
			&dataText,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.DeletedAt,
			&item.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan topology: %w", err)
		}
		item.Data = json.RawMessage(dataText)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topologies: %w", err)
	}
	return items, nil
}

func (r *Repository) Create(ctx context.Context, topology Topology) (Topology, error) {
	if r.db == nil {
		return Topology{}, apperrors.Internal("postgres is not configured")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Topology{}, fmt.Errorf("begin topology create: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM topologies
		WHERE project_id = $1 AND deleted_at IS NULL
	`, topology.ProjectID).Scan(&topology.Version); err != nil {
		return Topology{}, fmt.Errorf("next topology version: %w", err)
	}

	var dataText string
	err = tx.QueryRow(ctx, `
		INSERT INTO topologies (id, project_id, version, name, data, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5::jsonb, now(), now(), $6)
		RETURNING id::text, project_id::text, version, name, data::text, created_at, updated_at, deleted_at, created_by::text
	`, topology.ID, topology.ProjectID, topology.Version, topology.Name, string(topology.Data), topology.CreatedBy).Scan(
		&topology.ID,
		&topology.ProjectID,
		&topology.Version,
		&topology.Name,
		&dataText,
		&topology.CreatedAt,
		&topology.UpdatedAt,
		&topology.DeletedAt,
		&topology.CreatedBy,
	)
	if err != nil {
		return Topology{}, fmt.Errorf("create topology: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Topology{}, fmt.Errorf("commit topology create: %w", err)
	}

	topology.Data = json.RawMessage(dataText)
	return topology, nil
}

func (r *Repository) GetForOwner(ctx context.Context, topologyID, ownerID string) (Topology, error) {
	if r.db == nil {
		return Topology{}, apperrors.Internal("postgres is not configured")
	}

	var topology Topology
	var dataText string
	err := r.db.QueryRow(ctx, `
		SELECT t.id::text, t.project_id::text, t.version, t.name, t.data::text,
		       t.created_at, t.updated_at, t.deleted_at, t.created_by::text
		FROM topologies t
		INNER JOIN projects p ON p.id = t.project_id
		WHERE t.id = $1
		  AND p.owner_id = $2
		  AND t.deleted_at IS NULL
		  AND p.deleted_at IS NULL
	`, topologyID, ownerID).Scan(
		&topology.ID,
		&topology.ProjectID,
		&topology.Version,
		&topology.Name,
		&dataText,
		&topology.CreatedAt,
		&topology.UpdatedAt,
		&topology.DeletedAt,
		&topology.CreatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Topology{}, apperrors.NotFound("topology not found")
	}
	if err != nil {
		return Topology{}, fmt.Errorf("get topology: %w", err)
	}

	topology.Data = json.RawMessage(dataText)
	return topology, nil
}
