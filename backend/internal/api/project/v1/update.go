package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	projectv1 "github.com/netquest/netquest/backend/internal/contract/project/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Update обрабатывает PATCH /api/v1/projects/{projectId}: меняются только переданные поля.
// Ответы: 200 {project} / 400 кривой JSON / 404 / 422.
func (a *api) Update(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	var req projectv1.UpdateRequest
	if err := httpx.DecodeJSON(r, &req, a.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	project, err := a.projectService.Update(r.Context(), principal.UserID, r.PathValue(pathProjectID), converter.ToUpdateProjectInput(req))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, projectv1.ProjectResponse{Project: converter.ProjectToDTO(project)})
}
