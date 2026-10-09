package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	questv1 "github.com/netquest/netquest/backend/internal/contract/quest/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// GetAttempt обрабатывает GET /api/v1/quest-attempts/{attemptId}.
// Ответы: 200 {attempt} / 404 попытки нет или она чужая.
func (a *api) GetAttempt(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	attempt, err := a.questService.GetAttempt(r.Context(), principal.UserID, r.PathValue(pathAttemptID))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, questv1.AttemptResponse{Attempt: converter.AttemptToDTO(attempt)})
}
