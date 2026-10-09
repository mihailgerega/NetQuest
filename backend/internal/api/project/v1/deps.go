package v1

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// ProjectService — то, что обработчикам нужно от сервиса проектов.
// Реализация — *service из service/application/project. Мок: mocks/ (task mocks:gen).
type ProjectService interface {
	List(ctx context.Context, ownerID string) ([]model.Project, error)
	Create(ctx context.Context, ownerID string, in input.CreateProjectInput) (model.Project, error)
	Get(ctx context.Context, ownerID, projectID string) (model.Project, error)
	Update(ctx context.Context, ownerID, projectID string, in input.UpdateProjectInput) (model.Project, error)
	Delete(ctx context.Context, ownerID, projectID string) error
}
