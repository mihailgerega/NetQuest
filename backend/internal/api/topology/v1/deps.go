package v1

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// TopologyService — то, что обработчикам нужно от сервиса версий топологии.
// Реализация — *service из service/application/topology. Мок: mocks/ (task mocks:gen).
type TopologyService interface {
	ListForProject(ctx context.Context, ownerID, projectID string) ([]model.Topology, error)
	Create(ctx context.Context, ownerID, projectID string, in input.CreateTopologyInput) (model.Topology, model.ValidationResult, error)
	Get(ctx context.Context, ownerID, topologyID string) (model.Topology, error)
	ValidateStored(ctx context.Context, ownerID, topologyID string) (model.ValidationResult, error)
}
