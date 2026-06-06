package projects

import (
	"context"
	"strings"

	"github.com/netquest/netquest/backend/pkg/apperrors"
	"github.com/netquest/netquest/backend/pkg/idgen"
)

type Store interface {
	List(ctx context.Context, ownerID string) ([]Project, error)
	Create(ctx context.Context, project Project) (Project, error)
	GetByOwner(ctx context.Context, ownerID, projectID string) (Project, error)
	Update(ctx context.Context, ownerID, projectID string, req UpdateRequest) (Project, error)
	SoftDelete(ctx context.Context, ownerID, projectID string) error
	EnsureOwner(ctx context.Context, projectID, ownerID string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) List(ctx context.Context, ownerID string) ([]Project, error) {
	return s.store.List(ctx, ownerID)
}

func (s *Service) Create(ctx context.Context, ownerID string, req CreateRequest) (Project, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Project{}, apperrors.Validation("project name is required", nil)
	}
	if len(name) > 120 {
		return Project{}, apperrors.Validation("project name must be at most 120 characters", nil)
	}

	visibility := strings.TrimSpace(req.Visibility)
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	if !validVisibility(visibility) {
		return Project{}, apperrors.Validation("project visibility is invalid", map[string]any{
			"allowed": []string{VisibilityPrivate, VisibilityPublic, VisibilityUnlisted},
		})
	}

	id, err := idgen.NewUUID()
	if err != nil {
		return Project{}, err
	}

	return s.store.Create(ctx, Project{
		ID:          id,
		OwnerID:     ownerID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Visibility:  visibility,
	})
}

func (s *Service) Get(ctx context.Context, ownerID, projectID string) (Project, error) {
	return s.store.GetByOwner(ctx, ownerID, projectID)
}

func (s *Service) Update(ctx context.Context, ownerID, projectID string, req UpdateRequest) (Project, error) {
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return Project{}, apperrors.Validation("project name is required", nil)
		}
		if len(name) > 120 {
			return Project{}, apperrors.Validation("project name must be at most 120 characters", nil)
		}
		req.Name = &name
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		req.Description = &description
	}
	if req.Visibility != nil {
		visibility := strings.TrimSpace(*req.Visibility)
		if !validVisibility(visibility) {
			return Project{}, apperrors.Validation("project visibility is invalid", map[string]any{
				"allowed": []string{VisibilityPrivate, VisibilityPublic, VisibilityUnlisted},
			})
		}
		req.Visibility = &visibility
	}
	return s.store.Update(ctx, ownerID, projectID, req)
}

func (s *Service) Delete(ctx context.Context, ownerID, projectID string) error {
	return s.store.SoftDelete(ctx, ownerID, projectID)
}

func (s *Service) EnsureOwner(ctx context.Context, projectID, ownerID string) error {
	return s.store.EnsureOwner(ctx, projectID, ownerID)
}

func validVisibility(value string) bool {
	switch value {
	case VisibilityPrivate, VisibilityPublic, VisibilityUnlisted:
		return true
	default:
		return false
	}
}
