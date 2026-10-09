// Package v1 — API-слой аутентификации: обработчики /api/v1/auth/*.
//
// Поток запроса: net/http → цепочка middleware → ServeMux → хендлер →
// httpx.DecodeJSON (тело → authv1.*Request) → converter → service/application/auth
// → converter (model → authv1) → httpx.WriteJSON. Ошибку хендлер не переводит
// в HTTP-код сам — это делает httpx.WriteError.
//
// Refresh-токен живёт в httpOnly-cookie (cookie.go): хендлеры ставят её после
// входа и читают при refresh/logout, если токена нет в теле.
package v1

import "github.com/netquest/netquest/backend/internal/api/auditlog"

// Действия и ресурс аудита.
const (
	auditActionRegister    = "register"
	auditActionLogin       = "login"
	auditActionFailedLogin = "failed_login"
	auditActionLogout      = "logout"
	auditResourceUser      = "user"
)

// api — HTTP-обработчики аутентификации.
type api struct {
	// Интерфейс из deps.go (DIP): в unit-тестах подставляется мок.
	authService   AuthService
	auditRecorder auditlog.Recorder
	// jsonLimit — сколько байт тела читать при разборе JSON.
	jsonLimit int64
	// secureCookie — ставить refresh-cookie с флагом Secure (только HTTPS).
	secureCookie bool
}

// New создаёт обработчики аутентификации.
func New(authService AuthService, auditRecorder auditlog.Recorder, jsonLimit int64, secureCookie bool) *api {
	return &api{
		authService:   authService,
		auditRecorder: auditRecorder,
		jsonLimit:     jsonLimit,
		secureCookie:  secureCookie,
	}
}
