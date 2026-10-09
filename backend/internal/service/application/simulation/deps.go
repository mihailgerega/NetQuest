package simulation

import (
	"context"
	"time"

	"github.com/netquest/netquest/backend/internal/model"
)

// Интерфейсы зависимостей объявлены здесь, у потребителя (DIP). Моки: mocks/ (task mocks:gen).

// SimulationRepository — то, что сервису нужно от хранилища симуляций.
type SimulationRepository interface {
	Create(ctx context.Context, simulation model.Simulation) (model.Simulation, error)
	// UpdateStatus меняет статус; nil startedAt/finishedAt оставляют прежние значения.
	UpdateStatus(ctx context.Context, id string, status model.SimulationStatus, errorMessage *string, startedAt, finishedAt *time.Time) error
	InsertEvents(ctx context.Context, simulationID string, events []model.Event) error
	// GetForOwner — чужая или несуществующая симуляция → errs.ErrSimulationNotFound.
	GetForOwner(ctx context.Context, simulationID, ownerID string) (model.Simulation, error)
	ListEventsForOwner(ctx context.Context, simulationID, ownerID string) ([]model.Event, error)
}

// ProjectAuthorizer проверяет, что проект принадлежит пользователю.
type ProjectAuthorizer interface {
	EnsureOwner(ctx context.Context, projectID, ownerID string) error
}

// TopologyReader читает сохранённую версию топологии с проверкой владельца.
type TopologyReader interface {
	GetForOwner(ctx context.Context, topologyID, ownerID string) (model.Topology, error)
}

// SimulationEngine — движок симуляции (domain/engine).
type SimulationEngine interface {
	Run(ctx context.Context, req model.RunRequest) (model.RunResult, error)
}

// EventProducer публикует события симуляции для realtime-подписчиков (NATS).
type EventProducer interface {
	ProduceSimulationEvent(ctx context.Context, simulationID string, event model.Event) error
}
