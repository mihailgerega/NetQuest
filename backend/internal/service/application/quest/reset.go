package quest

import (
	"context"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
)

// Reset возвращает попытку в начало: статус in_progress, без проверок
// и открытых подсказок. Возвращает квест со статусом сброшенной попытки.
func (s *service) Reset(ctx context.Context, userID, attemptID string) (model.Quest, model.Attempt, error) {
	if err := s.ensureCatalog(ctx); err != nil {
		return model.Quest{}, model.Attempt{}, err
	}

	attempt, err := s.questRepository.ResetAttempt(ctx, attemptID, userID)
	if err != nil {
		return model.Quest{}, model.Attempt{}, err
	}

	quest, ok := s.findQuest(attempt.QuestID)
	if !ok {
		return model.Quest{}, model.Attempt{}, errs.ErrQuestNotFound
	}

	quest.AttemptStatus = attempt.Status

	return quest, attempt, nil
}
