package project

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/converter"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// List возвращает проекты владельца: сначала недавно изменённые.
func (r *repository) List(ctx context.Context, ownerID string) ([]model.Project, error) {
	const query = `
		SELECT ` + projectColumns + `
		FROM projects
		WHERE owner_id = $1 AND deleted_at IS NULL
		ORDER BY updated_at DESC, created_at DESC`

	rows, err := r.pool.Query(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("получить проекты: %w", err)
	}

	recs, err := pgx.CollectRows(rows, pgx.RowToStructByName[record.Project])
	if err != nil {
		return nil, fmt.Errorf("прочитать проекты: %w", err)
	}

	return converter.ProjectsToModel(recs), nil
}
