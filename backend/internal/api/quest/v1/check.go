package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	questv1 "github.com/netquest/netquest/backend/internal/contract/quest/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Check обрабатывает POST /api/v1/quest-attempts/{attemptId}/check:
// проверяет топологию-решение и сохраняет результат.
// Ответы: 200 {attempt, result} — и для пройденной, и для проваленной проверки /
// 400 кривой JSON / 404 попытки нет / 422 нет топологии.
func (a *api) Check(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	var req questv1.CheckRequest
	if err := httpx.DecodeJSON(r, &req, a.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	attempt, result, err := a.questService.Check(r.Context(), principal.UserID, r.PathValue(pathAttemptID), converter.ToCheckQuestInput(req))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	a.metrics.QuestChecksTotal.Add(1)

	action := "quest_checked"

	if result.Passed {
		a.metrics.QuestsCompletedTotal.Add(1)

		action = "quest_completed"
	}

	auditlog.Record(r, a.auditRecorder, &principal.UserID, action, auditResourceAttempt, attempt.ID)
	httpx.WriteJSON(w, http.StatusOK, questv1.CheckResponse{Attempt: converter.AttemptToDTO(attempt), Result: result})
}
