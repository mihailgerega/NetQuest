package user

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// FindByID ищет пользователя по ID — для Me и Refresh.
// Нет такого или удалён → errs.ErrUserNotFound.
func (r *repository) FindByID(ctx context.Context, id string) (model.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE id = $1 AND deleted_at IS NULL`

	return r.queryOne(ctx, query, id)
}
