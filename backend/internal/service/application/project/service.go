// Package project — сервисный слой проектов: список, создание, чтение,
// частичное обновление, мягкое удаление и проверка владельца.
//
// Место в цепочке: api/project/v1 → service/application/project → repository/project.
// Проверку владельца (EnsureOwner) используют и другие сервисы — топологий
// и симуляций, — прежде чем работать с данными проекта.
package project

import (
	"strings"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
)

// maxNameLength — предел длины имени проекта (в байтах).
const maxNameLength = 120

// service — сервис проектов. Неэкспортируемый тип: снаружи с ним работают
// через интерфейсы API-слоя и сервисов-потребителей.
type service struct {
	projectRepository ProjectRepository
}

// New создаёт сервис проектов.
func New(projectRepository ProjectRepository) *service {
	return &service{
		projectRepository: projectRepository,
	}
}

// validateName проверяет имя, уже обрезанное от пробелов.
func validateName(name string) error {
	if name == "" {
		return errs.NewValidationError("project name is required", nil)
	}

	if len(name) > maxNameLength {
		return errs.NewValidationError("project name must be at most 120 characters", nil)
	}

	return nil
}

// validateVisibility проверяет видимость; в ошибке перечислены допустимые значения.
func validateVisibility(visibility string) error {
	switch visibility {
	case model.VisibilityPrivate, model.VisibilityPublic, model.VisibilityUnlisted:
		return nil
	default:
		return errs.NewValidationError("project visibility is invalid", map[string]any{
			"allowed": model.ProjectVisibilities,
		})
	}
}

// trimmed возвращает строку без пробелов по краям.
func trimmed(value string) string {
	return strings.TrimSpace(value)
}
