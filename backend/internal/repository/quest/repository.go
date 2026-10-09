// Package quest — репозиторий квестов и попыток поверх PostgreSQL (pgx/v5).
//
// Место в цепочке: service/application/quest → repository/quest → таблицы
// quests (копия каталога из кода), quest_attempts и quest_check_results.
//
// Попытка видна только своему пользователю: каждый запрос к quest_attempts
// фильтрует по user_id, чужая и несуществующая неотличимы (errs.ErrAttemptNotFound).
package quest

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

// attemptColumns — колонки в порядке и под именами полей record.Attempt.
const attemptColumns = `id::text AS id, quest_id, user_id::text AS user_id, project_id::text AS project_id,
	current_topology_id::text AS current_topology_id, status, attempts_count, revealed_hints_count,
	last_check_result::text AS last_check_result, completed_at, created_at, updated_at`

// repository читает и пишет квесты и попытки через общий пул соединений.
type repository struct {
	pool *pgxpool.Pool
}

// New создаёт репозиторий квестов поверх пула.
func New(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}

// queryAttempt выполняет запрос одной попытки; «нет строки» → errs.ErrAttemptNotFound.
func (r *repository) queryAttempt(ctx context.Context, query string, args ...any) (model.Attempt, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return model.Attempt{}, fmt.Errorf("выполнить запрос попытки: %w", err)
	}

	rec, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[record.Attempt])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Attempt{}, errs.ErrAttemptNotFound
		}

		return model.Attempt{}, fmt.Errorf("прочитать попытку: %w", err)
	}

	return converter.AttemptToModel(rec), nil
}
