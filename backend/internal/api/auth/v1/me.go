package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	authv1 "github.com/netquest/netquest/backend/internal/contract/auth/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Me обрабатывает GET /api/v1/auth/me: профиль текущего пользователя.
// Ответы: 200 {user} / 401 без access-токена / 404 пользователь удалён.
func (a *api) Me(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	user, err := a.authService.Me(r.Context(), principal.UserID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, authv1.MeResponse{User: converter.UserToDTO(user)})
}
