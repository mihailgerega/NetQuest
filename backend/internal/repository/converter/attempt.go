package converter

import (
	"encoding/json"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// AttemptToModel переводит строку quest_attempts в доменную модель.
// last_check_result без проверок (NULL) остаётся nil.
func AttemptToModel(rec record.Attempt) model.Attempt {
	attempt := model.Attempt{
		ID:                 rec.ID,
		QuestID:            rec.QuestID,
		UserID:             rec.UserID,
		ProjectID:          rec.ProjectID,
		CurrentTopologyID:  rec.CurrentTopologyID,
		Status:             model.AttemptStatus(rec.Status),
		AttemptsCount:      rec.AttemptsCount,
		RevealedHintsCount: rec.RevealedHintsCount,
		CompletedAt:        rec.CompletedAt,
		CreatedAt:          rec.CreatedAt,
		UpdatedAt:          rec.UpdatedAt,
	}

	if rec.LastCheckResult != nil {
		attempt.LastCheckResult = json.RawMessage(*rec.LastCheckResult)
	}

	return attempt
}
