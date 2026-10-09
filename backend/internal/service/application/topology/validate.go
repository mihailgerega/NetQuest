package topology

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// ValidateStored заново валидирует сохранённую версию. Пригодится, когда
// правила валидатора стали строже, чем при сохранении версии.
func (s *service) ValidateStored(ctx context.Context, ownerID, topologyID string) (model.ValidationResult, error) {
	topology, err := s.topologyRepository.GetForOwner(ctx, topologyID, ownerID)
	if err != nil {
		return model.ValidationResult{}, err
	}

	return s.validator.ValidateRaw(topology.Data), nil
}
