package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/auth"
	questv1 "github.com/netquest/netquest/backend/internal/contract/quest/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Get обрабатывает GET /api/v1/quests/{questId} (ID или slug).
// Ответы: 200 {quest} / 404 квеста нет.
func (a *api) Get(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	quest, err := a.questService.Get(r.Context(), principal.UserID, r.PathValue(pathQuestID))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, questv1.QuestResponse{Quest: quest})
}
