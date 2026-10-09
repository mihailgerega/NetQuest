package project

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// Update записывает имя, описание и видимость проекта.
//
// Условие WHERE повторяет проверку владельца и удаления: если проект удалили
// между чтением в сервисе и этим UPDATE, строк не будет → errs.ErrProjectNotFound.
func (r *repository) Update(ctx context.Context, project model.Project) (model.Project, error) {
	const query = `
		UPDATE projects
		SET name = $3, description = NULLIF($4, ''), visibility = $5, updated_at = now()
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
		RETURNING ` + projectColumns

	return r.queryOne(ctx, query, project.ID, project.OwnerID, project.Name, project.Description, project.Visibility)
}
