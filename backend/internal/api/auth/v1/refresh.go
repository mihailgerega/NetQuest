package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/converter"
	authv1 "github.com/netquest/netquest/backend/internal/contract/auth/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Refresh обрабатывает POST /api/v1/auth/refresh: новая пара токенов
// по refresh-токену из тела или cookie.
// Ответы: 200 + новая refresh-cookie / 401 токена нет или он недействителен.
//
// Тело необязательно, поэтому ошибка его разбора игнорируется: токен тогда
// берётся из cookie.
func (a *api) Refresh(w http.ResponseWriter, r *http.Request) {
	var req authv1.RefreshRequest

	_ = httpx.DecodeJSON(r, &req, a.jsonLimit) //nolint:gosec // G104: тело необязательно, см. комментарий к хендлеру

	tokens, err := a.authService.Refresh(r.Context(), refreshTokenFrom(r, req.RefreshToken), clientInfo(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	a.setRefreshCookie(w, tokens.RefreshToken)
	httpx.WriteJSON(w, http.StatusOK, converter.AuthTokensToDTO(tokens))
}
