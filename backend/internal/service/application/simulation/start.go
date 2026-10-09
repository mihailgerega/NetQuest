package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// Start запускает сценарий над сохранённой версией топологии и возвращает
// запуск вместе с событиями и сводкой.
//
// Жизненный цикл строки simulations:
//
//	INSERT (pending) → UPDATE running + started_at → движок
//	→ INSERT событий → UPDATE completed/failed + finished_at
//
// Провал сети (нет маршрута, firewall...) — это completed-запуск движка со
// статусом failed: события сохраняются, клиент получает 201. А вот если
// движок отказался считать (топология невалидна), запуск помечается failed
// и клиент получает ошибку — 422 с подробностями валидации.
func (s *service) Start(ctx context.Context, userID string, in input.StartSimulationInput) (model.Simulation, model.RunResult, error) {
	topology, err := s.prepare(ctx, userID, in)
	if err != nil {
		return model.Simulation{}, model.RunResult{}, err
	}

	seed := time.Now().UTC().UnixNano()
	if in.Seed != nil {
		seed = *in.Seed
	}

	scenario, err := json.Marshal(in.Scenario)
	if err != nil {
		return model.Simulation{}, model.RunResult{}, fmt.Errorf("сериализовать сценарий: %w", err)
	}

	simulation, err := s.simulationRepository.Create(ctx, model.Simulation{
		ID:         uuid.NewString(),
		ProjectID:  in.ProjectID,
		TopologyID: in.TopologyID,
		UserID:     userID,
		Status:     model.SimulationStatusPending,
		Scenario:   scenario,
		Seed:       seed,
	})
	if err != nil {
		return model.Simulation{}, model.RunResult{}, fmt.Errorf("создать симуляцию: %w", err)
	}

	startedAt := time.Now().UTC()
	if err := s.simulationRepository.UpdateStatus(ctx, simulation.ID, model.SimulationStatusRunning, nil, &startedAt, nil); err != nil {
		return model.Simulation{}, model.RunResult{}, fmt.Errorf("отметить старт симуляции: %w", err)
	}

	s.metrics.SimulationsStartedTotal.Add(1)

	run, err := s.engine.Run(ctx, model.RunRequest{
		SimulationID: simulation.ID,
		Topology:     topology.Data,
		Scenario:     in.Scenario,
		Seed:         seed,
	})
	if err != nil {
		return model.Simulation{}, model.RunResult{}, s.markRunFailed(ctx, simulation.ID, err)
	}

	simulation, err = s.finish(ctx, simulation, run, startedAt)
	if err != nil {
		return model.Simulation{}, model.RunResult{}, err
	}

	s.publishEvents(ctx, simulation.ID, run.Events)

	return simulation, run, nil
}

// prepare проверяет запрос и права: заданы проект, версия топологии и тип
// сценария; проект — пользователя; версия принадлежит этому проекту.
//
// ID проекта сравнивается строкой — как его прислал клиент и как его вернула
// база (канонический UUID в нижнем регистре).
func (s *service) prepare(ctx context.Context, userID string, in input.StartSimulationInput) (model.Topology, error) {
	switch {
	case in.ProjectID == "":
		return model.Topology{}, errs.NewValidationError("projectId is required", nil)
	case in.TopologyID == "":
		return model.Topology{}, errs.NewValidationError("topologyId is required", nil)
	case in.Scenario.Type == "":
		return model.Topology{}, errs.NewValidationError("scenario.type is required", nil)
	}

	if err := s.projectAuthorizer.EnsureOwner(ctx, in.ProjectID, userID); err != nil {
		return model.Topology{}, err
	}

	topology, err := s.topologyReader.GetForOwner(ctx, in.TopologyID, userID)
	if err != nil {
		return model.Topology{}, err
	}

	if topology.ProjectID != in.ProjectID {
		return model.Topology{}, errs.NewValidationError("topology does not belong to project", nil)
	}

	return topology, nil
}

// markRunFailed помечает запуск failed, когда движок отказался считать,
// и возвращает ошибку для клиента.
//
// Статус пишется с context.WithoutCancel: если клиент уже отключился, ctx
// запроса отменён, а строка не должна навсегда остаться в running.
// Ошибку записи только логируем — клиенту важнее причина отказа движка.
func (s *service) markRunFailed(ctx context.Context, simulationID string, runErr error) error {
	message := runErr.Error()
	finishedAt := time.Now().UTC()

	if err := s.simulationRepository.UpdateStatus(context.WithoutCancel(ctx), simulationID, model.SimulationStatusFailed, &message, nil, &finishedAt); err != nil {
		slog.WarnContext(ctx, "не удалось отметить провал симуляции", "simulation_id", simulationID, "error", err)
	}

	var invalid errs.TopologyInvalidError
	if errors.As(runErr, &invalid) {
		return errs.NewValidationError("topology is invalid", invalid.Validation)
	}

	return fmt.Errorf("рассчитать симуляцию: %w", runErr)
}

// finish сохраняет события и итоговый статус и возвращает запуск в том виде,
// в каком он теперь лежит в базе. Текст ошибки провала — первая ошибка сводки.
func (s *service) finish(ctx context.Context, simulation model.Simulation, run model.RunResult, startedAt time.Time) (model.Simulation, error) {
	if err := s.simulationRepository.InsertEvents(ctx, simulation.ID, run.Events); err != nil {
		return model.Simulation{}, fmt.Errorf("сохранить события симуляции: %w", err)
	}

	finishedAt := time.Now().UTC()

	var errorMessage *string

	if run.Status == model.SimulationStatusFailed {
		message := "simulation failed"
		if len(run.Summary.Errors) > 0 {
			message = run.Summary.Errors[0]
		}

		errorMessage = &message
	}

	if err := s.simulationRepository.UpdateStatus(ctx, simulation.ID, run.Status, errorMessage, nil, &finishedAt); err != nil {
		return model.Simulation{}, fmt.Errorf("сохранить итог симуляции: %w", err)
	}

	simulation.Status = run.Status
	simulation.StartedAt = &startedAt
	simulation.FinishedAt = &finishedAt
	simulation.ErrorMessage = errorMessage

	if run.Status == model.SimulationStatusFailed {
		s.metrics.SimulationsFailedTotal.Add(1)
	} else {
		s.metrics.SimulationsCompletedTotal.Add(1)
	}

	return simulation, nil
}

// publishEvents публикует события в NATS по одному. Ошибка публикации
// не проваливает запрос: события уже сохранены, и клиент получит их в ответе
// и по WebSocket. Только лог.
func (s *service) publishEvents(ctx context.Context, simulationID string, events []model.Event) {
	for _, event := range events {
		if err := s.eventProducer.ProduceSimulationEvent(ctx, simulationID, event); err != nil {
			slog.WarnContext(ctx, "не удалось опубликовать событие симуляции", "simulation_id", simulationID, "error", err)
		}
	}
}
