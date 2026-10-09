package middleware

import (
	"net/http"
	"strings"

	"github.com/netquest/netquest/backend/internal/auth"
	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/httpx"
	"github.com/netquest/netquest/backend/internal/model"
)

// bearerPrefix — схема токена в заголовке Authorization (RFC 6750).
// Сравнивается с учётом регистра и ровно с одним пробелом.
const bearerPrefix = "Bearer "

// AccessTokenParser — то, что middleware нужно от JWT: проверить токен
// и узнать, кому он выдан. Реализация — *security.JWTManager.
type AccessTokenParser interface {
	ParseAccessToken(raw string) (model.Principal, error)
}

// RequireAuth пропускает только запросы с действительным access-токеном
// в заголовке Authorization: Bearer <JWT> и кладёт пользователя в ctx
// (его достаёт auth.PrincipalFromContext).
//
// Токен проверяется локально, без похода в базу: подпись, срок и издатель.
// Причины отказа клиенту не уточняются — любой кривой токен даёт один ответ 401.
func RequireAuth(parser AccessTokenParser) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			if header == "" {
				httpx.WriteError(w, r, errs.ErrBearerTokenRequired)
				return
			}

			token, ok := strings.CutPrefix(header, bearerPrefix)
			if !ok || strings.TrimSpace(token) == "" {
				httpx.WriteError(w, r, errs.ErrBearerTokenInvalid)
				return
			}

			principal, err := parser.ParseAccessToken(strings.TrimSpace(token))
			if err != nil {
				httpx.WriteError(w, r, errs.ErrBearerTokenInvalid)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), principal)))
		})
	}
}
