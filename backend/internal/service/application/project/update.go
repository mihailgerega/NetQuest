package project

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// Update частично обновляет проект владельца (PATCH): меняются только
// переданные поля.
//
// Шаги: проверить и обрезать переданные поля → прочитать текущий проект →
// наложить изменения → UPDATE. Сначала валидация: на кривой запрос база
// не трогается вовсе.
func (s *service) Update(ctx context.Context, ownerID, projectID string, in input.UpdateProjectInput) (model.Project, error) {
	patch, err := normalizePatch(in)
	if err != nil {
		return model.Project{}, err
	}

	project, err := s.projectRepository.GetByOwner(ctx, ownerID, projectID)
	if err != nil {
		return model.Project{}, err
	}

	if patch.Name != nil {
		project.Name = *patch.Name
	}

	if patch.Description != nil {
		project.Description = *patch.Description
	}

	if patch.Visibility != nil {
		project.Visibility = *patch.Visibility
	}

	return s.projectRepository.Update(ctx, project)
}

// normalizePatch обрезает переданные поля и проверяет их по тем же правилам,
// что и при создании. Описание может стать пустым — тогда оно очищается.
func normalizePatch(in input.UpdateProjectInput) (input.UpdateProjectInput, error) {
	var patch input.UpdateProjectInput

	if in.Name != nil {
		name := trimmed(*in.Name)
		if err := validateName(name); err != nil {
			return patch, err
		}

		patch.Name = &name
	}

	if in.Description != nil {
		description := trimmed(*in.Description)
		patch.Description = &description
	}

	if in.Visibility != nil {
		visibility := trimmed(*in.Visibility)
		if err := validateVisibility(visibility); err != nil {
			return patch, err
		}

		patch.Visibility = &visibility
	}

	return patch, nil
}
