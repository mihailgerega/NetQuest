package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
)

// uniqueViolation — SQLSTATE «нарушено ограничение уникальности» (класс 23,
// integrity constraint violation). Код стабилен между версиями PostgreSQL,
// в отличие от текста ошибки, который ещё и переводится на язык сервера.
const uniqueViolation = "23505"

// Create сохраняет нового пользователя и возвращает строку, как её записала база
// (created_at и updated_at ставит now()).
//
// Занятый email → errs.ErrEmailTaken. Узнаём это по коду ошибки PostgreSQL,
// а не предварительным SELECT: проверка и вставка двумя запросами оставили бы
// окно, в которое успеет второй Register с тем же email.
//
// NULLIF превращает пустые строки в NULL: у пользователя без пароля (demo)
// и без имени в колонках лежит NULL, а не пустая строка.
func (r *repository) Create(ctx context.Context, user model.User) (model.User, error) {
	const query = `
		INSERT INTO users (id, email, password_hash, display_name, role, created_at, updated_at)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, now(), now())
		RETURNING ` + userColumns

	created, err := r.queryOne(ctx, query, user.ID, user.Email, user.PasswordHash, user.DisplayName, user.Role)
	if err != nil {
		// errors.As ищет в цепочке *pgconn.PgError — ошибку, пришедшую от сервера БД
		// (в отличие от сетевых ошибок и отмены ctx, у которых SQLSTATE нет).
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return model.User{}, errs.ErrEmailTaken
		}

		return model.User{}, fmt.Errorf("создать пользователя: %w", err)
	}

	return created, nil
}
