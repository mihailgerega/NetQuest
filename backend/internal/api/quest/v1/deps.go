package v1

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// QuestService — то, что обработчикам нужно от сервиса квестов.
// Реализация — *service из service/application/quest. Мок: mocks/ (task mocks:gen).
type QuestService interface {
	List(ctx context.Context, userID string) ([]model.Quest, error)
	Get(ctx context.Context, userID, questID string) (model.Quest, error)
	Start(ctx context.Context, userID, questID string) (model.Quest, model.Attempt, error)
	GetAttempt(ctx context.Context, userID, attemptID string) (model.Attempt, error)
	Check(ctx context.Context, userID, attemptID string, in input.CheckQuestInput) (model.Attempt, model.QuestResult, error)
	Reset(ctx context.Context, userID, attemptID string) (model.Quest, model.Attempt, error)
	RevealHint(ctx context.Context, userID, attemptID string, revealedHintsCount int) (model.Attempt, error)
}
