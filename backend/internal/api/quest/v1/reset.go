package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	questv1 "github.com/netquest/netquest/backend/internal/contract/quest/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Reset обрабатывает POST /api/v1/quest-attempts/{attemptId}/reset.
// Ответы: 200 {quest, attempt} / 404.
func (a *api) Reset(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	quest, attempt, err := a.questService.Reset(r.Context(), principal.UserID, r.PathValue(pathAttemptID))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	auditlog.Record(r, a.auditRecorder, &principal.UserID, "quest_reset", auditResourceAttempt, attempt.ID)
	httpx.WriteJSON(w, http.StatusOK, questv1.QuestAttemptResponse{Quest: quest, Attempt: converter.AttemptToDTO(attempt)})
}
