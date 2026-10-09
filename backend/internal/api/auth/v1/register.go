package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/api/converter"
	authv1 "github.com/netquest/netquest/backend/internal/contract/auth/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Register обрабатывает POST /api/v1/auth/register.
// Ответы: 201 {user, accessToken, ...} + refresh-cookie / 400 кривой JSON /
// 409 email занят / 422 email или пароль не прошли проверку.
func (a *api) Register(w http.ResponseWriter, r *http.Request) {
	var req authv1.RegisterRequest
	if err := httpx.DecodeJSON(r, &req, a.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	tokens, err := a.authService.Register(r.Context(), converter.ToRegisterInput(req), clientInfo(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	a.setRefreshCookie(w, tokens.RefreshToken)
	auditlog.Record(r, a.auditRecorder, &tokens.User.ID, auditActionRegister, auditResourceUser, tokens.User.ID)
	httpx.WriteJSON(w, http.StatusCreated, converter.AuthTokensToDTO(tokens))
}
