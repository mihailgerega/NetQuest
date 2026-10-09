package topology

import (
	"context"
	"encoding/json"

	"github.com/netquest/netquest/backend/internal/model"
)

// Интерфейсы зависимостей объявлены здесь, у потребителя (DIP). Моки: mocks/ (task mocks:gen).

// TopologyRepository — то, что сервису нужно от хранилища версий топологии.
type TopologyRepository interface {
	ListForProjectOwner(ctx context.Context, projectID, ownerID string) ([]model.Topology, error)
	Create(ctx context.Context, topology model.Topology) (model.Topology, error)
	// GetForOwner — чужая или несуществующая версия → errs.ErrTopologyNotFound.
	GetForOwner(ctx context.Context, topologyID, ownerID string) (model.Topology, error)
}

// ProjectAuthorizer проверяет, что проект принадлежит пользователю.
// Реализация — сервис проектов; чужой проект → errs.ErrProjectNotFound.
type ProjectAuthorizer interface {
	EnsureOwner(ctx context.Context, projectID, ownerID string) error
}

// TopologyValidator — то, что сервису нужно от валидатора топологии.
type TopologyValidator interface {
	ValidateRaw(data json.RawMessage) model.ValidationResult
}
