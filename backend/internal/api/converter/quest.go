package converter

import (
	questv1 "github.com/netquest/netquest/backend/internal/contract/quest/v1"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// ToCheckQuestInput переводит запрос проверки решения во вход сервиса.
func ToCheckQuestInput(req questv1.CheckRequest) input.CheckQuestInput {
	return input.CheckQuestInput{
		Topology: req.Topology,
		Seed:     req.Seed,
	}
}

// AttemptToDTO переводит попытку в DTO ответа.
func AttemptToDTO(attempt model.Attempt) questv1.Attempt {
	return questv1.Attempt{
		ID:                 attempt.ID,
		QuestID:            attempt.QuestID,
		UserID:             attempt.UserID,
		ProjectID:          attempt.ProjectID,
		CurrentTopologyID:  attempt.CurrentTopologyID,
		Status:             attempt.Status,
		AttemptsCount:      attempt.AttemptsCount,
		RevealedHintsCount: attempt.RevealedHintsCount,
		LastCheckResult:    attempt.LastCheckResult,
		CompletedAt:        attempt.CompletedAt,
		CreatedAt:          attempt.CreatedAt,
		UpdatedAt:          attempt.UpdatedAt,
	}
}
