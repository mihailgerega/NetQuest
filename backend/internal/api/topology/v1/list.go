package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	topologyv1 "github.com/netquest/netquest/backend/internal/contract/topology/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// ListForProject обрабатывает GET /api/v1/projects/{projectId}/topologies.
// Ответы: 200 {topologies} — от новых версий к старым / 404 проекта нет или он чужой.
func (a *api) ListForProject(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	topologies, err := a.topologyService.ListForProject(r.Context(), principal.UserID, r.PathValue(pathProjectID))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, topologyv1.ListResponse{Topologies: converter.TopologiesToDTO(topologies)})
}
