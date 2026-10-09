// Package middleware — HTTP middleware NetQuest API: ID запроса, перехват паник,
// security-заголовки, CORS, лимит тела, таймаут, лог запросов и аутентификация.
//
// Rate limiter и счётчик метрик — тоже middleware, но живут рядом со своим
// состоянием: internal/ratelimit и internal/metrics.
package middleware

import "net/http"

// Middleware оборачивает обработчик: решает, звать ли следующий, и может
// изменить запрос или ответ.
type Middleware func(http.Handler) http.Handler

// Chain оборачивает handler в middleware так, что первый в списке — внешний:
// Chain(h, a, b) = a(b(h)). Запрос проходит a → b → h, ответ — в обратном порядке.
func Chain(handler http.Handler, middleware ...Middleware) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}

	return handler
}
