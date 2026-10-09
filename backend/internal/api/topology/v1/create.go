package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	topologyv1 "github.com/netquest/netquest/backend/internal/contract/topology/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// CreateForProject обрабатывает POST /api/v1/projects/{projectId}/topologies:
// сохраняет новую версию топологии.
// Ответы: 201 {topology, validation} / 400 кривой JSON / 404 чужой проект /
// 422 пустое имя или топология с ошибками (список — в error.details).
func (a *api) CreateForProject(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	var req topologyv1.CreateRequest
	if err := httpx.DecodeJSON(r, &req, a.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	topology, validation, err := a.topologyService.Create(r.Context(), principal.UserID, r.PathValue(pathProjectID), converter.ToCreateTopologyInput(req))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	auditlog.Record(r, a.auditRecorder, &principal.UserID, "topology.created", "topology", topology.ID)
	httpx.WriteJSON(w, http.StatusCreated, topologyv1.CreateResponse{
		Topology:   converter.TopologyToDTO(topology),
		Validation: validation,
	})
}
