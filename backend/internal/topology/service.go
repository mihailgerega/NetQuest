package topology

import (
	"context"
	"strings"

	"github.com/netquest/netquest/backend/pkg/apperrors"
	"github.com/netquest/netquest/backend/pkg/idgen"
)

type Store interface {
	ListForProjectOwner(ctx context.Context, projectID, ownerID string) ([]Topology, error)
	Create(ctx context.Context, topology Topology) (Topology, error)
	GetForOwner(ctx context.Context, topologyID, ownerID string) (Topology, error)
}

type ProjectAuthorizer interface {
	EnsureOwner(ctx context.Context, projectID, ownerID string) error
}

type Service struct {
	store     Store
	projects  ProjectAuthorizer
	validator Validator
}

type CreateResult struct {
	Topology   Topology         `json:"topology"`
	Validation ValidationResult `json:"validation"`
}

func NewService(store Store, projects ProjectAuthorizer, validator Validator) *Service {
	return &Service{store: store, projects: projects, validator: validator}
}

func (s *Service) ListForProject(ctx context.Context, ownerID, projectID string) ([]Topology, error) {
	if err := s.projects.EnsureOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}
	return s.store.ListForProjectOwner(ctx, projectID, ownerID)
}

func (s *Service) Create(ctx context.Context, ownerID, projectID string, req CreateRequest) (CreateResult, error) {
	if err := s.projects.EnsureOwner(ctx, projectID, ownerID); err != nil {
		return CreateResult{}, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return CreateResult{}, apperrors.Validation("topology name is required", nil)
	}
	if len(name) > 120 {
		return CreateResult{}, apperrors.Validation("topology name must be at most 120 characters", nil)
	}

	validation := s.validator.ValidateRaw(req.Data)
	if !validation.Valid {
		return CreateResult{}, apperrors.Validation("topology is invalid", validation)
	}

	id, err := idgen.NewUUID()
	if err != nil {
		return CreateResult{}, err
	}
	createdBy := ownerID
	topology, err := s.store.Create(ctx, Topology{
		ID:        id,
		ProjectID: projectID,
		Name:      name,
		Data:      req.Data,
		CreatedBy: &createdBy,
	})
	if err != nil {
		return CreateResult{}, err
	}

	return CreateResult{Topology: topology, Validation: validation}, nil
}

func (s *Service) Get(ctx context.Context, ownerID, topologyID string) (Topology, error) {
	return s.store.GetForOwner(ctx, topologyID, ownerID)
}

func (s *Service) ValidateStored(ctx context.Context, ownerID, topologyID string) (ValidationResult, error) {
	topology, err := s.store.GetForOwner(ctx, topologyID, ownerID)
	if err != nil {
		return ValidationResult{}, err
	}
	return s.validator.ValidateRaw(topology.Data), nil
}

func (s *Service) ValidateRaw(data []byte) ValidationResult {
	return s.validator.ValidateRaw(data)
}
