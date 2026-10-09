package topology

import (
	"context"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// ListForProject возвращает версии топологии проекта владельца.
// Чужой проект → errs.ErrProjectNotFound, а не пустой список: так клиент
// отличит «версий нет» от «проекта нет».
func (s *service) ListForProject(ctx context.Context, ownerID, projectID string) ([]model.Topology, error) {
	if err := s.projectAuthorizer.EnsureOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}

	topologies, err := s.topologyRepository.ListForProjectOwner(ctx, projectID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("получить версии топологии: %w", err)
	}

	return topologies, nil
}
