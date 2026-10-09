package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	topologyv1 "github.com/netquest/netquest/backend/internal/contract/topology/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Get обрабатывает GET /api/v1/topologies/{topologyId}.
// Ответы: 200 {topology} / 404 версии нет или проект чужой.
func (a *api) Get(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	topology, err := a.topologyService.Get(r.Context(), principal.UserID, r.PathValue(pathTopologyID))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, topologyv1.TopologyResponse{Topology: converter.TopologyToDTO(topology)})
}
