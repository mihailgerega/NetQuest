package topology

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// Get возвращает версию топологии владельца.
func (s *service) Get(ctx context.Context, ownerID, topologyID string) (model.Topology, error) {
	return s.topologyRepository.GetForOwner(ctx, topologyID, ownerID)
}
