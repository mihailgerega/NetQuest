package quest

import (
	"context"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
)

// RevealHint сохраняет, сколько прогрессивных подсказок открыл пользователь.
//
// Число приводится к допустимому: не меньше нуля, не больше числа подсказок
// квеста и не меньше уже открытого — счётчик только растёт, даже если клиент
// прислал меньшее значение (например, из устаревшей вкладки).
func (s *service) RevealHint(ctx context.Context, userID, attemptID string, revealedHintsCount int) (model.Attempt, error) {
	if err := s.ensureCatalog(ctx); err != nil {
		return model.Attempt{}, err
	}

	attempt, err := s.questRepository.GetAttemptForOwner(ctx, attemptID, userID)
	if err != nil {
		return model.Attempt{}, err
	}

	quest, ok := s.findQuest(attempt.QuestID)
	if !ok {
		return model.Attempt{}, errs.ErrQuestNotFound
	}

	count := max(revealedHintsCount, 0)

	if len(quest.ProgressiveHints) > 0 && count > len(quest.ProgressiveHints) {
		count = len(quest.ProgressiveHints)
	}

	count = max(count, attempt.RevealedHintsCount)

	return s.questRepository.UpdateRevealedHintsCount(ctx, attemptID, userID, count)
}
