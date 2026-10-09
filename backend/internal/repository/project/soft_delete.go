package project

import (
	"context"
	"fmt"

	errs "github.com/netquest/netquest/backend/internal/errors"
)

// SoftDelete помечает проект удалённым. Строка остаётся: на неё ссылаются
// версии топологии и симуляции.
//
// RowsAffected == 0 — проекта нет, он чужой или уже удалён → errs.ErrProjectNotFound.
func (r *repository) SoftDelete(ctx context.Context, ownerID, projectID string) error {
	const query = `
		UPDATE projects
		SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, projectID, ownerID)
	if err != nil {
		return fmt.Errorf("удалить проект: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return errs.ErrProjectNotFound
	}

	return nil
}
