package projects

import (
	"context"
	"testing"

	"github.com/netquest/netquest/backend/pkg/apperrors"
)

type fakeStore struct {
	projects map[string]Project
}

func (f fakeStore) List(ctx context.Context, ownerID string) ([]Project, error) {
	return nil, nil
}

func (f fakeStore) Create(ctx context.Context, project Project) (Project, error) {
	return project, nil
}

func (f fakeStore) GetByOwner(ctx context.Context, ownerID, projectID string) (Project, error) {
	project, ok := f.projects[projectID]
	if !ok || project.OwnerID != ownerID {
		return Project{}, apperrors.NotFound("project not found")
	}
	return project, nil
}

func (f fakeStore) Update(ctx context.Context, ownerID, projectID string, req UpdateRequest) (Project, error) {
	project, err := f.GetByOwner(ctx, ownerID, projectID)
	if err != nil {
		return Project{}, err
	}
	return project, nil
}

func (f fakeStore) SoftDelete(ctx context.Context, ownerID, projectID string) error {
	_, err := f.GetByOwner(ctx, ownerID, projectID)
	return err
}

func (f fakeStore) EnsureOwner(ctx context.Context, projectID, ownerID string) error {
	_, err := f.GetByOwner(ctx, ownerID, projectID)
	return err
}

func TestEnsureOwnerRejectsOtherUsersProject(t *testing.T) {
	service := NewService(fakeStore{projects: map[string]Project{
		"project-1": {ID: "project-1", OwnerID: "owner-1", Name: "Demo", Visibility: VisibilityPrivate},
	}})

	if err := service.EnsureOwner(context.Background(), "project-1", "owner-2"); err == nil {
		t.Fatal("expected ownership check to reject another user's project")
	}
}

func TestCreateValidatesVisibility(t *testing.T) {
	service := NewService(fakeStore{projects: map[string]Project{}})
	_, err := service.Create(context.Background(), "owner-1", CreateRequest{Name: "Demo", Visibility: "invalid"})
	if err == nil {
		t.Fatal("expected invalid visibility error")
	}
}
