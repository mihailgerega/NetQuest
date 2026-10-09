package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	projectv1 "github.com/netquest/netquest/backend/internal/contract/project/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Get обрабатывает GET /api/v1/projects/{projectId}.
// Ответы: 200 {project} / 404 проекта нет или он чужой.
func (a *api) Get(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	project, err := a.projectService.Get(r.Context(), principal.UserID, r.PathValue(pathProjectID))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, projectv1.ProjectResponse{Project: converter.ProjectToDTO(project)})
}
