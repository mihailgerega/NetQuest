package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/auth"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Validate обрабатывает POST /api/v1/topologies/{topologyId}/validate:
// повторная валидация сохранённой версии.
// Ответы: 200 — результат валидации как есть ({valid, errors}) / 404.
func (a *api) Validate(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	result, err := a.topologyService.ValidateStored(r.Context(), principal.UserID, r.PathValue(pathTopologyID))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, result)
}
