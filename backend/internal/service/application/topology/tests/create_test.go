package tests

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	topologyService "github.com/netquest/netquest/backend/internal/service/application/topology"
	"github.com/netquest/netquest/backend/internal/service/application/topology/mocks"
	"github.com/netquest/netquest/backend/internal/service/domain/validator"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// Unit-тесты сохранения версии топологии: владелец проекта, имя, валидация
// документа. Хранилище и проверка владельца — моки, валидатор настоящий.

const (
	ownerID   = "owner-1"
	projectID = "project-1"
	validDoc  = `{"nodes":[{"id":"c","type":"client"}],"links":[]}`
)

// TestCreate: валидная версия сохраняется с автором; ошибки имени и документа —
// 422 без похода в базу; чужой проект — 404.
func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		in         input.CreateTopologyInput
		ownerErr   error
		wantSaved  bool
		wantErr    error
		wantDetail bool // в 422 должны быть подробности валидации
	}{
		{name: "валидная версия", in: input.CreateTopologyInput{Name: " v1 ", Data: json.RawMessage(validDoc)}, wantSaved: true},
		{name: "чужой проект", in: input.CreateTopologyInput{Name: "v1", Data: json.RawMessage(validDoc)}, ownerErr: errs.ErrProjectNotFound, wantErr: errs.ErrProjectNotFound},
		{name: "пустое имя", in: input.CreateTopologyInput{Name: " ", Data: json.RawMessage(validDoc)}},
		{name: "длинное имя", in: input.CreateTopologyInput{Name: strings.Repeat("x", 121), Data: json.RawMessage(validDoc)}},
		{name: "невалидный документ", in: input.CreateTopologyInput{Name: "v1", Data: json.RawMessage(`{"nodes":[]}`)}, wantDetail: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := mocks.NewTopologyRepository(t)
			projects := mocks.NewProjectAuthorizer(t)
			svc := topologyService.New(repo, projects, validator.New())

			projects.EXPECT().EnsureOwner(ctx, projectID, ownerID).Return(tc.ownerErr).Once()

			if tc.wantSaved {
				repo.EXPECT().Create(ctx, mock.Anything).
					RunAndReturn(func(_ context.Context, topology model.Topology) (model.Topology, error) {
						assert.Equal(t, "v1", topology.Name)
						require.NotNil(t, topology.CreatedBy)
						assert.Equal(t, ownerID, *topology.CreatedBy)

						return topology, nil
					}).Once()
			}

			_, validation, err := svc.Create(ctx, ownerID, projectID, tc.in)

			switch {
			case tc.wantSaved:
				require.NoError(t, err)
				assert.True(t, validation.Valid)
			case tc.wantErr != nil:
				assert.ErrorIs(t, err, tc.wantErr)
			default:
				var validationErr *errs.ValidationError
				require.ErrorAs(t, err, &validationErr)

				if tc.wantDetail {
					assert.IsType(t, model.ValidationResult{}, validationErr.Details)
				}
			}
		})
	}
}
