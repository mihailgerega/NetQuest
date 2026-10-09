package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	questv1 "github.com/netquest/netquest/backend/internal/contract/quest/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// RevealHint обрабатывает POST /api/v1/quest-attempts/{attemptId}/reveal-hint:
// сохраняет, сколько прогрессивных подсказок открыто.
// Ответы: 200 {attempt} / 400 кривой JSON / 404.
func (a *api) RevealHint(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	var req questv1.RevealHintRequest
	if err := httpx.DecodeJSON(r, &req, a.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	attempt, err := a.questService.RevealHint(r.Context(), principal.UserID, r.PathValue(pathAttemptID), req.RevealedHintsCount)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	auditlog.Record(r, a.auditRecorder, &principal.UserID, "quest_hint_revealed", auditResourceAttempt, attempt.ID)
	httpx.WriteJSON(w, http.StatusOK, questv1.AttemptResponse{Attempt: converter.AttemptToDTO(attempt)})
}
