// Package auth — передача аутентифицированного пользователя через context.Context.
//
// middleware.RequireAuth проверяет access-токен и кладёт Principal в ctx запроса;
// API-хендлеры достают его отсюда. Ключ контекста — неэкспортируемый тип:
// значение под ним может положить и прочитать только этот пакет, чужой код
// случайно его не перезапишет.
package auth

import (
	"context"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
)

// contextKey — тип ключа контекста, уникальный для пакета.
type contextKey string

const principalKey contextKey = "principal"

// WithPrincipal возвращает дочерний контекст с пользователем.
// context.WithValue не меняет исходный ctx — новый нужно передать дальше явно.
func WithPrincipal(ctx context.Context, principal model.Principal) context.Context {
	return context.WithValue(ctx, principalKey, principal)
}

// PrincipalFromContext достаёт пользователя; ok=false — запрос не прошёл RequireAuth.
func PrincipalFromContext(ctx context.Context) (model.Principal, bool) {
	principal, ok := ctx.Value(principalKey).(model.Principal)
	return principal, ok
}

// RequirePrincipal — PrincipalFromContext для хендлеров за RequireAuth:
// без пользователя возвращает errs.ErrAuthenticationRequired (401).
func RequirePrincipal(ctx context.Context) (model.Principal, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return model.Principal{}, errs.ErrAuthenticationRequired
	}

	return principal, nil
}
