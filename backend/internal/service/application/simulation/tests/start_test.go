package tests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/metrics"
	"github.com/netquest/netquest/backend/internal/model"
	simulationService "github.com/netquest/netquest/backend/internal/service/application/simulation"
	"github.com/netquest/netquest/backend/internal/service/application/simulation/mocks"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// Unit-тесты запуска симуляции. Все зависимости — моки: проверяется порядок
// жизни строки simulations (pending → running → итог), обработка отказа движка
// и публикация событий.

const (
	userID     = "user-1"
	projectID  = "project-1"
	topologyID = "topology-1"
)

type deps struct {
	repo      *mocks.SimulationRepository
	projects  *mocks.ProjectAuthorizer
	topology  *mocks.TopologyReader
	engine    *mocks.SimulationEngine
	producer  *mocks.EventProducer
	counters  *metrics.Metrics
	startable interface {
		Start(ctx context.Context, userID string, in input.StartSimulationInput) (model.Simulation, model.RunResult, error)
	}
}

func newDeps(t *testing.T) deps {
	t.Helper()

	d := deps{
		repo:     mocks.NewSimulationRepository(t),
		projects: mocks.NewProjectAuthorizer(t),
		topology: mocks.NewTopologyReader(t),
		engine:   mocks.NewSimulationEngine(t),
		producer: mocks.NewEventProducer(t),
		counters: metrics.New(),
	}
	d.startable = simulationService.New(d.repo, d.projects, d.topology, d.engine, d.producer, d.counters)

	return d
}

func validInput() input.StartSimulationInput {
	seed := int64(42)

	return input.StartSimulationInput{
		ProjectID:  projectID,
		TopologyID: topologyID,
		Scenario:   model.Scenario{Type: model.ScenarioICMPPing, SourceNodeID: "client-1", Target: "server-1"},
		Seed:       &seed,
	}
}

// expectPrepared — проект свой, версия из этого проекта, строка создана и помечена running.
func (d deps) expectPrepared(ctx context.Context) {
	d.projects.EXPECT().EnsureOwner(ctx, projectID, userID).Return(nil).Once()
	d.topology.EXPECT().GetForOwner(ctx, topologyID, userID).
		Return(model.Topology{ID: topologyID, ProjectID: projectID, Data: json.RawMessage(`{"nodes":[],"links":[]}`)}, nil).Once()
	d.repo.EXPECT().Create(ctx, mock.Anything).
		RunAndReturn(func(_ context.Context, simulation model.Simulation) (model.Simulation, error) {
			return simulation, nil
		}).Once()
	d.repo.EXPECT().UpdateStatus(ctx, mock.Anything, model.SimulationStatusRunning, (*string)(nil), mock.Anything, (*time.Time)(nil)).Return(nil).Once()
}

// TestStartValidation: обязательные поля и принадлежность версии проекту.
func TestStartValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(in *input.StartSimulationInput)
		wantErr string
	}{
		{name: "нет проекта", mutate: func(in *input.StartSimulationInput) { in.ProjectID = "" }, wantErr: "projectId is required"},
		{name: "нет версии", mutate: func(in *input.StartSimulationInput) { in.TopologyID = "" }, wantErr: "topologyId is required"},
		{name: "нет типа сценария", mutate: func(in *input.StartSimulationInput) { in.Scenario.Type = "" }, wantErr: "scenario.type is required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := newDeps(t)
			in := validInput()
			tc.mutate(&in)

			_, _, err := d.startable.Start(context.Background(), userID, in)

			var validationErr *errs.ValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, tc.wantErr, validationErr.Message)
		})
	}
}

// TestStartTopologyFromOtherProject: версия чужого проекта пользователя — 422.
func TestStartTopologyFromOtherProject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	d := newDeps(t)

	d.projects.EXPECT().EnsureOwner(ctx, projectID, userID).Return(nil).Once()
	d.topology.EXPECT().GetForOwner(ctx, topologyID, userID).Return(model.Topology{ProjectID: "project-2"}, nil).Once()

	_, _, err := d.startable.Start(ctx, userID, validInput())

	var validationErr *errs.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "topology does not belong to project", validationErr.Message)
}

// TestStartSuccess: события сохраняются и публикуются, итог записан, счётчики увеличены.
func TestStartSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	d := newDeps(t)
	d.expectPrepared(ctx)

	run := model.RunResult{
		Status: model.SimulationStatusFailed,
		Events: []model.Event{{ID: "e1"}, {ID: "e2"}},
		Summary: model.Summary{
			Errors: []string{"no route from client-1 to server-1"},
		},
	}

	d.engine.EXPECT().Run(ctx, mock.MatchedBy(func(req model.RunRequest) bool { return req.Seed == 42 })).Return(run, nil).Once()
	d.repo.EXPECT().InsertEvents(ctx, mock.Anything, run.Events).Return(nil).Once()
	d.repo.EXPECT().UpdateStatus(ctx, mock.Anything, model.SimulationStatusFailed, mock.Anything, (*time.Time)(nil), mock.Anything).
		Run(func(_ context.Context, _ string, _ model.SimulationStatus, errorMessage *string, _, _ *time.Time) {
			require.NotNil(t, errorMessage)
			assert.Equal(t, "no route from client-1 to server-1", *errorMessage, "текст провала — первая ошибка сводки")
		}).Return(nil).Once()
	d.producer.EXPECT().ProduceSimulationEvent(ctx, mock.Anything, mock.Anything).Return(errors.New("NATS недоступен")).Times(2)

	simulation, result, err := d.startable.Start(ctx, userID, validInput())

	require.NoError(t, err, "ошибка публикации не проваливает запрос")
	assert.Equal(t, model.SimulationStatusFailed, simulation.Status)
	assert.NotNil(t, simulation.StartedAt)
	assert.NotNil(t, simulation.FinishedAt)
	assert.Equal(t, run, result)
	assert.Equal(t, int64(1), d.counters.SimulationsStartedTotal.Load())
	assert.Equal(t, int64(1), d.counters.SimulationsFailedTotal.Load())
}

// TestStartEngineRejectsTopology: отказ движка помечает запуск failed,
// а невалидная топология превращается в 422 с подробностями.
func TestStartEngineRejectsTopology(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	d := newDeps(t)
	d.expectPrepared(ctx)

	validation := model.ValidationResult{Valid: false, Errors: []model.ValidationError{{Path: "nodes", Message: "nodes field is required"}}}

	d.engine.EXPECT().Run(ctx, mock.Anything).Return(model.RunResult{}, errs.TopologyInvalidError{Validation: validation}).Once()
	d.repo.EXPECT().UpdateStatus(mock.Anything, mock.Anything, model.SimulationStatusFailed, mock.Anything, (*time.Time)(nil), mock.Anything).
		Run(func(_ context.Context, _ string, _ model.SimulationStatus, errorMessage *string, _, _ *time.Time) {
			assert.Equal(t, "topology is invalid", *errorMessage)
		}).Return(nil).Once()

	_, _, err := d.startable.Start(ctx, userID, validInput())

	var validationErr *errs.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "topology is invalid", validationErr.Message)
	assert.Equal(t, validation, validationErr.Details)
	assert.Equal(t, int64(0), d.counters.SimulationsFailedTotal.Load(), "отказ движка не считается проваленной симуляцией")
}
