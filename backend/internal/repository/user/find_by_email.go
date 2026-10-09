package user

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// FindByEmail ищет пользователя по email — для Login. Email сравнивается
// как есть: нормализует его (нижний регистр, без пробелов) сервис.
// Нет такого или удалён → errs.ErrUserNotFound.
func (r *repository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE email = $1 AND deleted_at IS NULL`

	return r.queryOne(ctx, query, email)
}
