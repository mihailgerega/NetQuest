package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	projectv1 "github.com/netquest/netquest/backend/internal/contract/project/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// List обрабатывает GET /api/v1/projects. Ответ: 200 {projects}.
func (a *api) List(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	projects, err := a.projectService.List(r.Context(), principal.UserID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, projectv1.ListResponse{Projects: converter.ProjectsToDTO(projects)})
}
