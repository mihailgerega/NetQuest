package quest

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
)

// Start начинает новую попытку квеста. Прежние попытки не трогаются:
// статус квеста дальше берётся из самой свежей.
func (s *service) Start(ctx context.Context, userID, questID string) (model.Quest, model.Attempt, error) {
	if err := s.ensureCatalog(ctx); err != nil {
		return model.Quest{}, model.Attempt{}, err
	}

	quest, ok := s.findQuest(questID)
	if !ok {
		return model.Quest{}, model.Attempt{}, errs.ErrQuestNotFound
	}

	attempt, err := s.questRepository.CreateAttempt(ctx, model.Attempt{
		ID:            uuid.NewString(),
		QuestID:       quest.ID,
		UserID:        userID,
		Status:        model.AttemptInProgress,
		AttemptsCount: 0,
	})
	if err != nil {
		return model.Quest{}, model.Attempt{}, fmt.Errorf("создать попытку: %w", err)
	}

	quest.AttemptStatus = attempt.Status

	return quest, attempt, nil
}
