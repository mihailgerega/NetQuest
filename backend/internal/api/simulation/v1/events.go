package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/auth"
	simulationv1 "github.com/netquest/netquest/backend/internal/contract/simulation/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Events обрабатывает GET /api/v1/simulations/{simulationId}/events —
// сохранённые события по порядку; через него фронтенд опрашивает Timeline.
// Ответ: 200 {events} (для чужой симуляции — пустой список).
func (a *api) Events(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	events, err := a.simulationService.Events(r.Context(), principal.UserID, r.PathValue(pathSimulationID))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, simulationv1.EventsResponse{Events: events})
}
