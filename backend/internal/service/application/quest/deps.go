package quest

import (
	"context"
	"encoding/json"
	"time"

	"github.com/netquest/netquest/backend/internal/model"
)

// Интерфейсы зависимостей объявлены здесь, у потребителя (DIP). Моки: mocks/ (task mocks:gen).

// QuestRepository — то, что сервису нужно от хранилища квестов и попыток.
// Методы попыток ищут попытку пользователя: чужая → errs.ErrAttemptNotFound.
type QuestRepository interface {
	UpsertQuests(ctx context.Context, quests []model.Quest) error
	LatestAttemptStatuses(ctx context.Context, userID string) (map[string]model.AttemptStatus, error)
	CreateAttempt(ctx context.Context, attempt model.Attempt) (model.Attempt, error)
	GetAttemptForOwner(ctx context.Context, attemptID, userID string) (model.Attempt, error)
	// UpdateAttemptAfterCheck: nil completedAt оставляет прежнее время прохождения.
	UpdateAttemptAfterCheck(
		ctx context.Context,
		attemptID, userID string,
		status model.AttemptStatus,
		result model.QuestResult,
		completedAt *time.Time,
	) (model.Attempt, error)
	InsertCheckResult(ctx context.Context, id, attemptID, userID string, result model.QuestResult) error
	ResetAttempt(ctx context.Context, attemptID, userID string) (model.Attempt, error)
	// UpdateRevealedHintsCount не уменьшает счётчик (GREATEST в SQL).
	UpdateRevealedHintsCount(ctx context.Context, attemptID, userID string, count int) (model.Attempt, error)
}

// QuestChecker проверяет решение квеста (domain/checker).
type QuestChecker interface {
	Check(ctx context.Context, quest model.Quest, data json.RawMessage, seed int64) model.QuestResult
}
