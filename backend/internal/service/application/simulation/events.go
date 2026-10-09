package simulation

import (
	"context"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// Events возвращает события запуска по порядку. Для чужого запуска — пустой
// список: существование проверяют Get или сам клиент, у которого есть ID.
func (s *service) Events(ctx context.Context, userID, simulationID string) ([]model.Event, error) {
	events, err := s.simulationRepository.ListEventsForOwner(ctx, simulationID, userID)
	if err != nil {
		return nil, fmt.Errorf("получить события симуляции: %w", err)
	}

	return events, nil
}
