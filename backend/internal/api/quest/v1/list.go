package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/auth"
	questv1 "github.com/netquest/netquest/backend/internal/contract/quest/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// List обрабатывает GET /api/v1/quests: каталог со статусами попыток пользователя.
// Ответ: 200 {quests}.
func (a *api) List(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	quests, err := a.questService.List(r.Context(), principal.UserID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, questv1.ListResponse{Quests: quests})
}
