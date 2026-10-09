package topology

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// Create сохраняет новую версию топологии проекта и возвращает её вместе
// с результатом валидации.
//
// Шаги: проверить владельца проекта → проверить имя → провалидировать документ
// (невалидный → 422 со списком проблем в details) → INSERT с номером версии max+1.
func (s *service) Create(
	ctx context.Context,
	ownerID, projectID string,
	in input.CreateTopologyInput,
) (model.Topology, model.ValidationResult, error) {
	if err := s.projectAuthorizer.EnsureOwner(ctx, projectID, ownerID); err != nil {
		return model.Topology{}, model.ValidationResult{}, err
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		return model.Topology{}, model.ValidationResult{}, errs.NewValidationError("topology name is required", nil)
	}

	if len(name) > maxNameLength {
		return model.Topology{}, model.ValidationResult{}, errs.NewValidationError("topology name must be at most 120 characters", nil)
	}

	validation := s.validator.ValidateRaw(in.Data)
	if !validation.Valid {
		return model.Topology{}, model.ValidationResult{}, errs.NewValidationError("topology is invalid", validation)
	}

	createdBy := ownerID

	topology, err := s.topologyRepository.Create(ctx, model.Topology{
		ID:        uuid.NewString(),
		ProjectID: projectID,
		Name:      name,
		Data:      in.Data,
		CreatedBy: &createdBy,
	})
	if err != nil {
		return model.Topology{}, model.ValidationResult{}, fmt.Errorf("сохранить версию топологии: %w", err)
	}

	return topology, validation, nil
}
