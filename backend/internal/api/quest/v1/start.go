package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	questv1 "github.com/netquest/netquest/backend/internal/contract/quest/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Start обрабатывает POST /api/v1/quests/{questId}/start: новая попытка квеста.
// Ответы: 201 {quest, attempt} / 404 квеста нет.
func (a *api) Start(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	quest, attempt, err := a.questService.Start(r.Context(), principal.UserID, r.PathValue(pathQuestID))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	a.metrics.QuestsStartedTotal.Add(1)
	auditlog.Record(r, a.auditRecorder, &principal.UserID, "quest_started", auditResourceQuest, quest.ID)
	httpx.WriteJSON(w, http.StatusCreated, questv1.QuestAttemptResponse{Quest: quest, Attempt: converter.AttemptToDTO(attempt)})
}
