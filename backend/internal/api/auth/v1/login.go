package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/api/converter"
	authv1 "github.com/netquest/netquest/backend/internal/contract/auth/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Login обрабатывает POST /api/v1/auth/login.
// Ответы: 200 + refresh-cookie / 400 кривой JSON / 401 неверный email или пароль.
// Неудачный вход тоже попадает в аудит — по нему видно подбор паролей.
func (a *api) Login(w http.ResponseWriter, r *http.Request) {
	var req authv1.LoginRequest
	if err := httpx.DecodeJSON(r, &req, a.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	tokens, err := a.authService.Login(r.Context(), converter.ToLoginInput(req), clientInfo(r))
	if err != nil {
		auditlog.Record(r, a.auditRecorder, nil, auditActionFailedLogin, auditResourceUser, "")
		httpx.WriteError(w, r, err)

		return
	}

	a.setRefreshCookie(w, tokens.RefreshToken)
	auditlog.Record(r, a.auditRecorder, &tokens.User.ID, auditActionLogin, auditResourceUser, tokens.User.ID)
	httpx.WriteJSON(w, http.StatusOK, converter.AuthTokensToDTO(tokens))
}
