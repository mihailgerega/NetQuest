package project

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// Get возвращает проект владельца; чужой или несуществующий → errs.ErrProjectNotFound.
func (s *service) Get(ctx context.Context, ownerID, projectID string) (model.Project, error) {
	return s.projectRepository.GetByOwner(ctx, ownerID, projectID)
}
