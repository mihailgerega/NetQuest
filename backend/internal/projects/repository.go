package projects

import (
	"context"
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

func (r *Repository) List(ctx context.Context, ownerID string) ([]Project, error) {
	if r.db == nil {
		return nil, apperrors.Internal("postgres is not configured")
	}

	rows, err := r.db.Query(ctx, `
		SELECT id::text, owner_id::text, name, COALESCE(description, ''), visibility, created_at, updated_at, deleted_at
		FROM projects
		WHERE owner_id = $1 AND deleted_at IS NULL
		ORDER BY updated_at DESC, created_at DESC
	`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	items := make([]Project, 0)
	for rows.Next() {
		var project Project
		if err := rows.Scan(
			&project.ID,
			&project.OwnerID,
			&project.Name,
			&project.Description,
			&project.Visibility,
			&project.CreatedAt,
			&project.UpdatedAt,
			&project.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		items = append(items, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return items, nil
}

func (r *Repository) Create(ctx context.Context, project Project) (Project, error) {
	if r.db == nil {
		return Project{}, apperrors.Internal("postgres is not configured")
	}

	err := r.db.QueryRow(ctx, `
		INSERT INTO projects (id, owner_id, name, description, visibility, created_at, updated_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, now(), now())
		RETURNING id::text, owner_id::text, name, COALESCE(description, ''), visibility, created_at, updated_at, deleted_at
	`, project.ID, project.OwnerID, project.Name, project.Description, project.Visibility).Scan(
		&project.ID,
		&project.OwnerID,
		&project.Name,
		&project.Description,
		&project.Visibility,
		&project.CreatedAt,
		&project.UpdatedAt,
		&project.DeletedAt,
	)
	if err != nil {
		return Project{}, fmt.Errorf("create project: %w", err)
	}

	return project, nil
}

func (r *Repository) GetByOwner(ctx context.Context, ownerID, projectID string) (Project, error) {
	if r.db == nil {
		return Project{}, apperrors.Internal("postgres is not configured")
	}

	var project Project
	err := r.db.QueryRow(ctx, `
		SELECT id::text, owner_id::text, name, COALESCE(description, ''), visibility, created_at, updated_at, deleted_at
		FROM projects
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
	`, projectID, ownerID).Scan(
		&project.ID,
		&project.OwnerID,
		&project.Name,
		&project.Description,
		&project.Visibility,
		&project.CreatedAt,
		&project.UpdatedAt,
		&project.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, apperrors.NotFound("project not found")
	}
	if err != nil {
		return Project{}, fmt.Errorf("get project: %w", err)
	}
	return project, nil
}

func (r *Repository) Update(ctx context.Context, ownerID, projectID string, req UpdateRequest) (Project, error) {
	if r.db == nil {
		return Project{}, apperrors.Internal("postgres is not configured")
	}

	current, err := r.GetByOwner(ctx, ownerID, projectID)
	if err != nil {
		return Project{}, err
	}
	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.Description != nil {
		current.Description = *req.Description
	}
	if req.Visibility != nil {
		current.Visibility = *req.Visibility
	}

	err = r.db.QueryRow(ctx, `
		UPDATE projects
		SET name = $3, description = NULLIF($4, ''), visibility = $5, updated_at = now()
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
		RETURNING id::text, owner_id::text, name, COALESCE(description, ''), visibility, created_at, updated_at, deleted_at
	`, projectID, ownerID, current.Name, current.Description, current.Visibility).Scan(
		&current.ID,
		&current.OwnerID,
		&current.Name,
		&current.Description,
		&current.Visibility,
		&current.CreatedAt,
		&current.UpdatedAt,
		&current.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, apperrors.NotFound("project not found")
	}
	if err != nil {
		return Project{}, fmt.Errorf("update project: %w", err)
	}
	return current, nil
}

func (r *Repository) SoftDelete(ctx context.Context, ownerID, projectID string) error {
	if r.db == nil {
		return apperrors.Internal("postgres is not configured")
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE projects
		SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
	`, projectID, ownerID)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NotFound("project not found")
	}
	return nil
}

func (r *Repository) EnsureOwner(ctx context.Context, projectID, ownerID string) error {
	_, err := r.GetByOwner(ctx, ownerID, projectID)
	return err
}
