package project

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// GetByOwner читает проект владельца. Чужой, удалённый или несуществующий
// проект → errs.ErrProjectNotFound.
func (r *repository) GetByOwner(ctx context.Context, ownerID, projectID string) (model.Project, error) {
	const query = `
		SELECT ` + projectColumns + `
		FROM projects
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL`

	return r.queryOne(ctx, query, projectID, ownerID)
}
