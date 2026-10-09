package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/auth"
	authv1 "github.com/netquest/netquest/backend/internal/contract/auth/v1"
	commonv1 "github.com/netquest/netquest/backend/internal/contract/common/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// Logout обрабатывает POST /api/v1/auth/logout: отзывает refresh-токен
// и стирает cookie. Ответ: 200 {"ok": true} — даже если токена не было.
//
// Маршрут без RequireAuth: выйти можно и с истёкшим access-токеном. Поэтому
// пользователь в ctx обычно не найден, и запись аудита идёт без него.
func (a *api) Logout(w http.ResponseWriter, r *http.Request) {
	var req authv1.LogoutRequest

	_ = httpx.DecodeJSON(r, &req, a.jsonLimit) //nolint:gosec // G104: тело необязательно, см. комментарий к хендлеру

	if err := a.authService.Logout(r.Context(), refreshTokenFrom(r, req.RefreshToken)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	a.clearRefreshCookie(w)

	var userID *string
	if principal, ok := auth.PrincipalFromContext(r.Context()); ok {
		userID = &principal.UserID
	}

	auditlog.Record(r, a.auditRecorder, userID, auditActionLogout, auditResourceUser, "")
	httpx.WriteJSON(w, http.StatusOK, commonv1.OKResponse{OK: true})
}
