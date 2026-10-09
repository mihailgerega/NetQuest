// Package user — репозиторий пользователей поверх PostgreSQL (pgx/v5).
//
// Место в цепочке: service/application/auth → repository/user → таблица users.
// Наружу отдаёт только доменные типы и доменные ошибки: pgx.ErrNoRows
// и unique_violation превращаются в errs.* здесь, выше pgx не протекает.
//
// Удалённые пользователи (deleted_at IS NOT NULL) для поиска не существуют.
package user

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

// userColumns — колонки в порядке и под именами полей record.User.
// Одна константа для всех SELECT и RETURNING: добавили колонку — она появилась везде.
const userColumns = `id::text AS id, email, COALESCE(display_name, '') AS display_name,
	COALESCE(avatar_url, '') AS avatar_url, role, created_at, updated_at, deleted_at,
	COALESCE(password_hash, '') AS password_hash`

// repository читает и пишет пользователей через общий пул соединений.
type repository struct {
	// Пул создаёт и закрывает DI-контейнер; репозиторий им только пользуется.
	pool *pgxpool.Pool
}

// New создаёт репозиторий пользователей поверх пула.
func New(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}

// queryOne выполняет запрос, возвращающий одного пользователя, и переводит
// «нет строки» в errs.ErrUserNotFound. Общая часть всех методов: они отличаются
// только SQL и параметрами.
func (r *repository) queryOne(ctx context.Context, query string, args ...any) (model.User, error) {
	// Значения идут параметрами $1, $2..., а не склейкой строки: так исключена SQL-инъекция.
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return model.User{}, fmt.Errorf("выполнить запрос пользователя: %w", err)
	}

	// CollectOneRow читает ровно одну строку по тегам db и закрывает rows;
	// ноль строк → pgx.ErrNoRows.
	rec, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[record.User])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, errs.ErrUserNotFound
		}

		return model.User{}, fmt.Errorf("прочитать пользователя: %w", err)
	}

	return converter.UserToModel(rec), nil
}
