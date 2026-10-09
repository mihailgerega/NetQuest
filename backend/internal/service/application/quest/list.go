package quest

import (
	"context"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// List возвращает весь каталог со статусом последней попытки пользователя
// по каждому квесту.
func (s *service) List(ctx context.Context, userID string) ([]model.Quest, error) {
	if err := s.ensureCatalog(ctx); err != nil {
		return nil, err
	}

	statuses, err := s.questRepository.LatestAttemptStatuses(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("получить статусы попыток: %w", err)
	}

	quests := cloneCatalog(s.catalog)
	for i := range quests {
		withAttemptStatus(&quests[i], statuses)
	}

	return quests, nil
}
