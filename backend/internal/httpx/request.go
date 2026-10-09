package httpx

import (
	"context"
	"net"
	"net/http"
	"strings"
)

// contextKey — тип ключа контекста, уникальный для пакета.
type contextKey string

const requestIDKey contextKey = "request_id"

// WithRequestID возвращает дочерний контекст с ID запроса.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestID достаёт ID запроса; пусто — запрос не прошёл middleware.RequestID.
func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

// ClientIP определяет IP клиента: первый адрес из X-Forwarded-For, затем
// X-Real-IP, затем адрес TCP-соединения.
//
// Заголовкам доверяем, потому что API работает за reverse proxy (Caddy),
// который их выставляет. Без прокси клиент может подставить любой IP —
// этим значением нельзя защищать доступ, только писать в аудит и считать лимиты.
func ClientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		first, _, _ := strings.Cut(forwardedFor, ",")
		if ip := strings.TrimSpace(first); ip != "" {
			return ip
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
