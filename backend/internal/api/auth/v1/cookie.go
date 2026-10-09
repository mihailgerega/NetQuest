package v1

import (
	"net/http"
	"time"

	"github.com/netquest/netquest/backend/internal/httpx"
	"github.com/netquest/netquest/backend/internal/service/input"
)

const (
	// RefreshTokenCookieName — имя cookie с refresh-токеном.
	RefreshTokenCookieName = "netquest_refresh_token"
	// refreshCookieMaxAge — срок cookie: 30 дней, как срок refresh-токена по умолчанию.
	refreshCookieMaxAge = 30 * 24 * time.Hour
)

// setRefreshCookie кладёт refresh-токен в cookie.
//
// HttpOnly — JavaScript страницы не видит токен, и XSS не может его украсть.
// SameSite=Lax — браузер не пошлёт cookie в запросах с чужих сайтов (кроме
// переходов по ссылке), это защита от CSRF. Secure — только по HTTPS (в production).
func (a *api) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: Secure задаётся конфигом — локально API работает по HTTP
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(refreshCookieMaxAge.Seconds()),
		HttpOnly: true,
		Secure:   a.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearRefreshCookie удаляет cookie: MaxAge < 0 велит браузеру стереть её сразу.
func (a *api) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: Secure задаётся конфигом, как у setRefreshCookie
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

// refreshTokenFrom берёт refresh-токен из тела, а если его там нет — из cookie.
func refreshTokenFrom(r *http.Request, fromBody string) string {
	if fromBody != "" {
		return fromBody
	}

	if cookie, err := r.Cookie(RefreshTokenCookieName); err == nil {
		return cookie.Value
	}

	return ""
}

// clientInfo — откуда пришёл запрос: User-Agent и IP клиента.
func clientInfo(r *http.Request) input.ClientInfo {
	return input.ClientInfo{
		UserAgent: r.UserAgent(),
		IPAddress: httpx.ClientIP(r),
	}
}
