package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	projectService "github.com/netquest/netquest/backend/internal/service/application/project"
	"github.com/netquest/netquest/backend/internal/service/application/project/mocks"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// Unit-тесты сервиса проектов: правила имени и видимости, частичное обновление,
// проверка владельца. Хранилище — мок.

const ownerID = "owner-1"

func strPtr(s string) *string { return &s }

// TestCreate: имя и описание обрезаются, видимость по умолчанию private;
// пустое, слишком длинное имя и неизвестная видимость — 422 до похода в базу.
func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		in             input.CreateProjectInput
		wantSaved      bool
		wantVisibility string
		wantErr        string
	}{
		{name: "видимость по умолчанию", in: input.CreateProjectInput{Name: "  Lab ", Description: " notes "}, wantSaved: true, wantVisibility: model.VisibilityPrivate},
		{name: "публичный", in: input.CreateProjectInput{Name: "Lab", Visibility: " public "}, wantSaved: true, wantVisibility: model.VisibilityPublic},
		{name: "пустое имя", in: input.CreateProjectInput{Name: "   "}, wantErr: "project name is required"},
		{name: "длинное имя", in: input.CreateProjectInput{Name: strings.Repeat("x", 121)}, wantErr: "project name must be at most 120 characters"},
		{name: "неизвестная видимость", in: input.CreateProjectInput{Name: "Lab", Visibility: "invalid"}, wantErr: "project visibility is invalid"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := mocks.NewProjectRepository(t)
			svc := projectService.New(repo)

			if tc.wantSaved {
				repo.EXPECT().Create(ctx, mock.Anything).
					RunAndReturn(func(_ context.Context, project model.Project) (model.Project, error) {
						assert.NotEmpty(t, project.ID)
						assert.Equal(t, ownerID, project.OwnerID)
						assert.Equal(t, strings.TrimSpace(tc.in.Name), project.Name)
						assert.Equal(t, strings.TrimSpace(tc.in.Description), project.Description)
						assert.Equal(t, tc.wantVisibility, project.Visibility)

						return project, nil
					}).Once()
			}

			_, err := svc.Create(ctx, ownerID, tc.in)

			if tc.wantErr != "" {
				var validationErr *errs.ValidationError
				require.ErrorAs(t, err, &validationErr)
				assert.Equal(t, tc.wantErr, validationErr.Message)

				return
			}

			require.NoError(t, err)
		})
	}
}

// TestVisibilityErrorListsAllowedValues: в details ошибки — допустимые значения.
func TestVisibilityErrorListsAllowedValues(t *testing.T) {
	t.Parallel()

	svc := projectService.New(mocks.NewProjectRepository(t))

	_, err := svc.Create(context.Background(), ownerID, input.CreateProjectInput{Name: "Lab", Visibility: "secret"})

	var validationErr *errs.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, map[string]any{"allowed": []string{"private", "public", "unlisted"}}, validationErr.Details)
}

// TestEnsureOwner: чужой проект не отличается от несуществующего.
func TestEnsureOwner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := mocks.NewProjectRepository(t)
	svc := projectService.New(repo)

	repo.EXPECT().GetByOwner(ctx, "owner-2", "project-1").Return(model.Project{}, errs.ErrProjectNotFound).Once()

	assert.ErrorIs(t, svc.EnsureOwner(ctx, "project-1", "owner-2"), errs.ErrProjectNotFound)
}

// TestUpdatePatch: меняются только переданные поля; кривое поле отсекается
// до чтения проекта.
func TestUpdatePatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := mocks.NewProjectRepository(t)
	svc := projectService.New(repo)

	current := model.Project{ID: "project-1", OwnerID: ownerID, Name: "Old", Description: "keep", Visibility: model.VisibilityPrivate}

	repo.EXPECT().GetByOwner(ctx, ownerID, "project-1").Return(current, nil).Once()
	repo.EXPECT().Update(ctx, mock.Anything).
		RunAndReturn(func(_ context.Context, project model.Project) (model.Project, error) {
			assert.Equal(t, "New", project.Name)
			assert.Equal(t, "keep", project.Description, "описание не передано — не меняется")
			assert.Equal(t, model.VisibilityPrivate, project.Visibility)

			return project, nil
		}).Once()

	_, err := svc.Update(ctx, ownerID, "project-1", input.UpdateProjectInput{Name: strPtr("  New  ")})
	require.NoError(t, err)

	_, err = svc.Update(ctx, ownerID, "project-1", input.UpdateProjectInput{Visibility: strPtr("")})
	var validationErr *errs.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "project visibility is invalid", validationErr.Message)
}
