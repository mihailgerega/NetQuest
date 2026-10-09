// Package project — репозиторий проектов поверх PostgreSQL (pgx/v5).
//
// Место в цепочке: service/application/project → repository/project → таблица projects.
//
// Каждый запрос фильтрует по owner_id: проект виден только владельцу, чужой
// и несуществующий неотличимы (errs.ErrProjectNotFound). Удаление мягкое —
// строки с deleted_at для всех методов как будто нет.
package project

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/converter"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// projectColumns — колонки в порядке и под именами полей record.Project.
const projectColumns = `id::text AS id, owner_id::text AS owner_id, name,
	COALESCE(description, '') AS description, visibility, created_at, updated_at, deleted_at`

// repository читает и пишет проекты через общий пул соединений.
type repository struct {
	pool *pgxpool.Pool
}

// New создаёт репозиторий проектов поверх пула.
func New(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}

// queryOne выполняет запрос одного проекта; «нет строки» → errs.ErrProjectNotFound.
func (r *repository) queryOne(ctx context.Context, query string, args ...any) (model.Project, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return model.Project{}, fmt.Errorf("выполнить запрос проекта: %w", err)
	}

	rec, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[record.Project])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Project{}, errs.ErrProjectNotFound
		}

		return model.Project{}, fmt.Errorf("прочитать проект: %w", err)
	}

	return converter.ProjectToModel(rec), nil
}
