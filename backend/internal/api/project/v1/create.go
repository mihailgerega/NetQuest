package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	projectv1 "github.com/netquest/netquest/backend/internal/contract/project/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Create обрабатывает POST /api/v1/projects.
// Ответы: 201 {project} / 400 кривой JSON / 422 имя или видимость не прошли проверку.
func (a *api) Create(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	var req projectv1.CreateRequest
	if err := httpx.DecodeJSON(r, &req, a.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	project, err := a.projectService.Create(r.Context(), principal.UserID, converter.ToCreateProjectInput(req))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	auditlog.Record(r, a.auditRecorder, &principal.UserID, auditActionCreated, auditResourceObject, project.ID)
	httpx.WriteJSON(w, http.StatusCreated, projectv1.ProjectResponse{Project: converter.ProjectToDTO(project)})
}
