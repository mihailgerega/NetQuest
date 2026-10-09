package project

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// ProjectRepository — то, что сервису нужно от хранилища проектов.
// Интерфейс объявлен здесь, у потребителя (DIP). Мок: mocks/ (task mocks:gen).
//
// Все методы, кроме Create, ищут проект владельца: чужой или удалённый →
// errs.ErrProjectNotFound.
type ProjectRepository interface {
	List(ctx context.Context, ownerID string) ([]model.Project, error)
	Create(ctx context.Context, project model.Project) (model.Project, error)
	GetByOwner(ctx context.Context, ownerID, projectID string) (model.Project, error)
	Update(ctx context.Context, project model.Project) (model.Project, error)
	SoftDelete(ctx context.Context, ownerID, projectID string) error
}
