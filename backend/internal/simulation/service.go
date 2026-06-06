package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/netquest/netquest/backend/internal/observability"
	"github.com/netquest/netquest/backend/internal/realtime"
	"github.com/netquest/netquest/backend/internal/topology"
	"github.com/netquest/netquest/backend/pkg/apperrors"
	"github.com/netquest/netquest/backend/pkg/idgen"
)

type Store interface {
	Create(ctx context.Context, simulation Simulation) (Simulation, error)
	UpdateStatus(ctx context.Context, id string, status Status, errorMessage *string, startedAt, finishedAt *time.Time) error
	InsertEvents(ctx context.Context, simulationID string, events []Event) error
	GetForOwner(ctx context.Context, simulationID, ownerID string) (Simulation, error)
	ListEventsForOwner(ctx context.Context, simulationID, ownerID string) ([]Event, error)
}

type ProjectAuthorizer interface {
	EnsureOwner(ctx context.Context, projectID, ownerID string) error
}

type TopologyStore interface {
	GetForOwner(ctx context.Context, topologyID, ownerID string) (topology.Topology, error)
}

type Service struct {
	store      Store
	projects   ProjectAuthorizer
	topologies TopologyStore
	engine     SimulationEngine
	publisher  realtime.EventPublisher
	logger     *slog.Logger
	metrics    *observability.Metrics
}

type StartResult struct {
	Simulation Simulation `json:"simulation"`
	Events     []Event    `json:"events"`
	Summary    Summary    `json:"summary"`
}

func NewService(store Store, projects ProjectAuthorizer, topologies TopologyStore, engine SimulationEngine, publisher realtime.EventPublisher, logger *slog.Logger, metrics *observability.Metrics) *Service {
	return &Service{store: store, projects: projects, topologies: topologies, engine: engine, publisher: publisher, logger: logger, metrics: metrics}
}

func (s *Service) Start(ctx context.Context, userID string, req StartRequest) (StartResult, error) {
	if req.ProjectID == "" {
		return StartResult{}, apperrors.Validation("projectId is required", nil)
	}
	if req.TopologyID == "" {
		return StartResult{}, apperrors.Validation("topologyId is required", nil)
	}
	if req.Scenario.Type == "" {
		return StartResult{}, apperrors.Validation("scenario.type is required", nil)
	}
	if err := s.projects.EnsureOwner(ctx, req.ProjectID, userID); err != nil {
		return StartResult{}, err
	}

	topologyRecord, err := s.topologies.GetForOwner(ctx, req.TopologyID, userID)
	if err != nil {
		return StartResult{}, err
	}
	if topologyRecord.ProjectID != req.ProjectID {
		return StartResult{}, apperrors.Validation("topology does not belong to project", nil)
	}

	id, err := idgen.NewUUID()
	if err != nil {
		return StartResult{}, err
	}
	seed := time.Now().UTC().UnixNano()
	if req.Seed != nil {
		seed = *req.Seed
	}
	scenarioBytes, err := json.Marshal(req.Scenario)
	if err != nil {
		return StartResult{}, fmt.Errorf("marshal simulation scenario: %w", err)
	}

	simulation, err := s.store.Create(ctx, Simulation{
		ID:         id,
		ProjectID:  req.ProjectID,
		TopologyID: req.TopologyID,
		UserID:     userID,
		Status:     StatusPending,
		Scenario:   scenarioBytes,
		Seed:       seed,
	})
	if err != nil {
		return StartResult{}, err
	}

	startedAt := time.Now().UTC()
	if err := s.store.UpdateStatus(ctx, simulation.ID, StatusRunning, nil, &startedAt, nil); err != nil {
		return StartResult{}, err
	}
	if s.metrics != nil {
		s.metrics.SimulationsStartedTotal.Add(1)
	}

	run, runErr := s.engine.Run(ctx, RunRequest{
		SimulationID: simulation.ID,
		Topology:     topologyRecord.Data,
		Scenario:     req.Scenario,
		Seed:         seed,
	})
	if runErr != nil {
		message := runErr.Error()
		finishedAt := time.Now().UTC()
		_ = s.store.UpdateStatus(context.Background(), simulation.ID, StatusFailed, &message, nil, &finishedAt)
		var invalid TopologyInvalidError
		if errors.As(runErr, &invalid) {
			return StartResult{}, apperrors.Validation("topology is invalid", invalid.Validation)
		}
		return StartResult{}, runErr
	}

	if err := s.store.InsertEvents(ctx, simulation.ID, run.Events); err != nil {
		return StartResult{}, err
	}

	finishedAt := time.Now().UTC()
	var errorMessage *string
	if run.Status == StatusFailed {
		message := "simulation failed"
		if len(run.Summary.Errors) > 0 {
			message = run.Summary.Errors[0]
		}
		errorMessage = &message
	}
	if err := s.store.UpdateStatus(ctx, simulation.ID, run.Status, errorMessage, nil, &finishedAt); err != nil {
		return StartResult{}, err
	}
	simulation.Status = run.Status
	simulation.StartedAt = &startedAt
	simulation.FinishedAt = &finishedAt
	simulation.ErrorMessage = errorMessage
	if s.metrics != nil {
		if run.Status == StatusFailed {
			s.metrics.SimulationsFailedTotal.Add(1)
		} else {
			s.metrics.SimulationsCompletedTotal.Add(1)
		}
	}

	s.publishEvents(ctx, simulation.ID, run.Events)
	return StartResult{Simulation: simulation, Events: run.Events, Summary: run.Summary}, nil
}

func (s *Service) Get(ctx context.Context, userID, simulationID string) (Simulation, error) {
	return s.store.GetForOwner(ctx, simulationID, userID)
}

func (s *Service) Events(ctx context.Context, userID, simulationID string) ([]Event, error) {
	return s.store.ListEventsForOwner(ctx, simulationID, userID)
}

func (s *Service) publishEvents(ctx context.Context, simulationID string, events []Event) {
	if s.publisher == nil {
		return
	}
	for _, event := range events {
		if err := s.publisher.Publish(ctx, "netquest.simulations."+simulationID+".events", map[string]any{
			"type":         "simulation.event",
			"simulationId": simulationID,
			"event":        event,
		}); err != nil && s.logger != nil {
			s.logger.Warn("publish simulation event failed", slog.String("error", err.Error()))
		}
	}
}
