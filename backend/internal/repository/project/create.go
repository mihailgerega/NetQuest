package project

import (
	"context"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// Create сохраняет новый проект и возвращает строку, как её записала база.
// Пустое описание хранится как NULL.
func (r *repository) Create(ctx context.Context, project model.Project) (model.Project, error) {
	const query = `
		INSERT INTO projects (id, owner_id, name, description, visibility, created_at, updated_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, now(), now())
		RETURNING ` + projectColumns

	created, err := r.queryOne(ctx, query, project.ID, project.OwnerID, project.Name, project.Description, project.Visibility)
	if err != nil {
		return model.Project{}, fmt.Errorf("создать проект: %w", err)
	}

	return created, nil
}
