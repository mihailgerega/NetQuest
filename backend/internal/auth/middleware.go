package auth

import (
	"net/http"
	"strings"

	"github.com/netquest/netquest/backend/internal/httpx"
	"github.com/netquest/netquest/backend/pkg/apperrors"
)

func RequireAuth(manager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			if header == "" {
				httpx.WriteError(w, r, apperrors.Unauthorized("authorization bearer token is required"))
				return
			}

			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || strings.TrimSpace(token) == "" {
				httpx.WriteError(w, r, apperrors.Unauthorized("authorization bearer token is invalid"))
				return
			}

			principal, err := manager.ParseAccessToken(strings.TrimSpace(token))
			if err != nil {
				httpx.WriteError(w, r, apperrors.Unauthorized("authorization bearer token is invalid"))
				return
			}

			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}
