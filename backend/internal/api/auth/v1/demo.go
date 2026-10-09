package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Demo обрабатывает POST /api/v1/auth/demo: вход в общий demo-аккаунт без тела.
// Ответы: 200 + refresh-cookie / 403 demo-вход выключен.
func (a *api) Demo(w http.ResponseWriter, r *http.Request) {
	tokens, err := a.authService.Demo(r.Context(), clientInfo(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	a.setRefreshCookie(w, tokens.RefreshToken)
	auditlog.Record(r, a.auditRecorder, &tokens.User.ID, auditActionLogin, auditResourceUser, tokens.User.ID)
	httpx.WriteJSON(w, http.StatusOK, converter.AuthTokensToDTO(tokens))
}
