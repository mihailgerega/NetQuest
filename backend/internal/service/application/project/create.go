package project

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// Create создаёт проект владельца. Имя и описание обрезаются от пробелов,
// видимость по умолчанию — private.
func (s *service) Create(ctx context.Context, ownerID string, in input.CreateProjectInput) (model.Project, error) {
	name := trimmed(in.Name)
	if err := validateName(name); err != nil {
		return model.Project{}, err
	}

	visibility := trimmed(in.Visibility)
	if visibility == "" {
		visibility = model.VisibilityPrivate
	}

	if err := validateVisibility(visibility); err != nil {
		return model.Project{}, err
	}

	project, err := s.projectRepository.Create(ctx, model.Project{
		ID:          uuid.NewString(),
		OwnerID:     ownerID,
		Name:        name,
		Description: trimmed(in.Description),
		Visibility:  visibility,
	})
	if err != nil {
		return model.Project{}, fmt.Errorf("сохранить проект: %w", err)
	}

	return project, nil
}
