package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	simulationv1 "github.com/netquest/netquest/backend/internal/contract/simulation/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Get обрабатывает GET /api/v1/simulations/{simulationId}.
// Ответы: 200 {simulation} / 404 симуляции нет или проект чужой.
func (a *api) Get(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	simulation, err := a.simulationService.Get(r.Context(), principal.UserID, r.PathValue(pathSimulationID))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, simulationv1.SimulationResponse{Simulation: converter.SimulationToDTO(simulation)})
}
