package quest

import (
	"context"
	"fmt"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
)

// Get возвращает квест по ID или slug со статусом попытки пользователя.
// Нет такого → errs.ErrQuestNotFound.
func (s *service) Get(ctx context.Context, userID, questID string) (model.Quest, error) {
	if err := s.ensureCatalog(ctx); err != nil {
		return model.Quest{}, err
	}

	quest, ok := s.findQuest(questID)
	if !ok {
		return model.Quest{}, errs.ErrQuestNotFound
	}

	statuses, err := s.questRepository.LatestAttemptStatuses(ctx, userID)
	if err != nil {
		return model.Quest{}, fmt.Errorf("получить статусы попыток: %w", err)
	}

	withAttemptStatus(&quest, statuses)

	return quest, nil
}
