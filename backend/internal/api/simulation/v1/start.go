package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	simulationv1 "github.com/netquest/netquest/backend/internal/contract/simulation/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Start обрабатывает POST /api/v1/simulations: запуск сценария над версией топологии.
// Ответы: 201 {simulation, events, summary} — и когда сеть доставила пакет,
// и когда нет (статус в simulation.status) / 400 кривой JSON /
// 404 проект или версия чужие / 422 нет полей, версия из другого проекта
// или топология не прошла валидацию.
func (a *api) Start(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	var req simulationv1.StartRequest
	if err := httpx.DecodeJSON(r, &req, a.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	simulation, run, err := a.simulationService.Start(r.Context(), principal.UserID, converter.ToStartSimulationInput(req))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	auditlog.Record(r, a.auditRecorder, &principal.UserID, "simulation.started", "simulation", simulation.ID)
	httpx.WriteJSON(w, http.StatusCreated, simulationv1.StartResponse{
		Simulation: converter.SimulationToDTO(simulation),
		Events:     run.Events,
		Summary:    run.Summary,
	})
}
