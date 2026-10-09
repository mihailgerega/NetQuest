package simulation

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// Get возвращает запуск пользователя; чужой или несуществующий →
// errs.ErrSimulationNotFound.
func (s *service) Get(ctx context.Context, userID, simulationID string) (model.Simulation, error) {
	return s.simulationRepository.GetForOwner(ctx, simulationID, userID)
}
