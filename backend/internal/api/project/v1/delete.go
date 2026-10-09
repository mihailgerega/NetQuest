package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/auth"
	commonv1 "github.com/netquest/netquest/backend/internal/contract/common/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Delete обрабатывает DELETE /api/v1/projects/{projectId} (мягкое удаление).
// Ответы: 200 {"ok": true} / 404.
func (a *api) Delete(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	projectID := r.PathValue(pathProjectID)

	if err := a.projectService.Delete(r.Context(), principal.UserID, projectID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	auditlog.Record(r, a.auditRecorder, &principal.UserID, auditActionDeleted, auditResourceObject, projectID)
	httpx.WriteJSON(w, http.StatusOK, commonv1.OKResponse{OK: true})
}
