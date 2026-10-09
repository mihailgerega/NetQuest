package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// requestIDHeader — заголовок с ID запроса: клиент может прислать свой
// (например, ID трассировки фронтенда), иначе сервис сгенерирует UUID.
const requestIDHeader = "X-Request-ID"

// RequestID кладёт ID запроса в ctx и возвращает его клиенту в заголовке:
// по нему запрос находится в логах, а клиент указывает его в обращениях.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(requestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		w.Header().Set(requestIDHeader, requestID)
		next.ServeHTTP(w, r.WithContext(httpx.WithRequestID(r.Context(), requestID)))
	})
}

// Recover перехватывает панику обработчика: пишет её со стеком в лог
// и отвечает 500, вместо того чтобы net/http оборвал соединение.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.ErrorContext(r.Context(), "паника в обработчике HTTP-запроса",
					slog.Any("panic", recovered),
					slog.String("request_id", httpx.RequestID(r.Context())),
					slog.String("stack", string(debug.Stack())),
				)
				httpx.WriteError(w, r, errs.ErrUnexpected)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RequestTimeout ограничивает срок context.Context запроса: по нему
// отменяются запросы в PostgreSQL и Redis, если ответ считается слишком долго.
//
// Сам ответ таймаут не обрывает: обработчик, который не смотрит в ctx,
// доработает до конца.
func RequestTimeout(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// BodyLimit ограничивает размер тела запроса: http.MaxBytesReader вернёт
// ошибку при чтении сверх limit, и разбор JSON завершится 400.
// 0 и меньше — без ограничения.
func BodyLimit(limit int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limit > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger пишет в лог одну запись на запрос: метод, путь, статус, длительность.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		slog.InfoContext(r.Context(), "HTTP-запрос",
			slog.String("request_id", httpx.RequestID(r.Context())),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("remote_addr", r.RemoteAddr),
		)
	})
}

// statusRecorder запоминает код ответа для лога. Обработчик, который не звал
// WriteHeader, ответил 200 — отсюда значение по умолчанию.
//
// Обёртка видна обработчикам ниже по цепочке как http.ResponseWriter и только:
// дополнительные интерфейсы исходного writer'а (http.Hijacker, http.Flusher)
// через неё не видны. Из-за этого upgrade WebSocket в /api/v1/ws не проходит
// (ответ 500 «websocket hijack unsupported»), и фронтенд работает через опрос.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader запоминает код и передаёт его дальше.
func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
