package quest

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// defaultCheckSeed — seed проверки, если клиент его не прислал.
const defaultCheckSeed = 7

// Check проверяет решение попытки и сохраняет результат.
//
// Шаги: найти попытку пользователя → найти её квест → проверить топологию
// чекером → записать результат в историю → обновить попытку (статус, +1 проверка,
// последний результат, время прохождения при успехе).
//
// Порядок важен: попытка ищется до проверки тела, поэтому чужая попытка с пустой
// топологией даёт 404, а не 422.
func (s *service) Check(ctx context.Context, userID, attemptID string, in input.CheckQuestInput) (model.Attempt, model.QuestResult, error) {
	if err := s.ensureCatalog(ctx); err != nil {
		return model.Attempt{}, model.QuestResult{}, err
	}

	attempt, err := s.questRepository.GetAttemptForOwner(ctx, attemptID, userID)
	if err != nil {
		return model.Attempt{}, model.QuestResult{}, err
	}

	quest, ok := s.findQuest(attempt.QuestID)
	if !ok {
		return model.Attempt{}, model.QuestResult{}, errs.ErrQuestNotFound
	}

	if len(in.Topology) == 0 {
		return model.Attempt{}, model.QuestResult{}, errs.NewValidationError("topology is required", nil)
	}

	seed := int64(defaultCheckSeed)
	if in.Seed != nil {
		seed = *in.Seed
	}

	result := s.checker.Check(ctx, quest, in.Topology, seed)

	status := model.AttemptFailed

	var completedAt *time.Time

	if result.Passed {
		status = model.AttemptCompleted
		now := time.Now().UTC()
		completedAt = &now
	}

	if err := s.questRepository.InsertCheckResult(ctx, uuid.NewString(), attempt.ID, userID, result); err != nil {
		return model.Attempt{}, model.QuestResult{}, fmt.Errorf("сохранить результат проверки: %w", err)
	}

	attempt, err = s.questRepository.UpdateAttemptAfterCheck(ctx, attempt.ID, userID, status, result, completedAt)
	if err != nil {
		return model.Attempt{}, model.QuestResult{}, err
	}

	return attempt, result, nil
}
