package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// TestWriteError: каждая ошибка сервиса превращается в свой HTTP-статус и код,
// текст — ровно текст из контракта, даже если ошибку обернули через %w.
// Неизвестные ошибки — 500 без подробностей.
func TestWriteError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantCode    string
		wantMessage string
		wantDetails any
	}{
		{"не найдено (обёрнуто)", fmt.Errorf("получить проект: %w", errs.ErrProjectNotFound), http.StatusNotFound, "not_found", "project not found", nil},
		{"нет токена", errs.ErrBearerTokenRequired, http.StatusUnauthorized, "unauthorized", "authorization bearer token is required", nil},
		{"demo выключен", errs.ErrDemoAuthDisabled, http.StatusForbidden, "forbidden", "demo auth is disabled", nil},
		{"email занят", errs.ErrEmailTaken, http.StatusConflict, "conflict", "email is already registered", nil},
		{"пустое тело", errs.ErrRequestBodyRequired, http.StatusBadRequest, "bad_request", "request body is required", nil},
		{"кривой JSON", &errs.InvalidJSONError{Cause: errors.New("unexpected EOF")}, http.StatusBadRequest, "bad_request", "invalid JSON body: unexpected EOF", nil},
		{"валидация с подробностями", errs.NewValidationError("project visibility is invalid", map[string]any{"allowed": []any{"private"}}), http.StatusUnprocessableEntity, "validation_failed", "project visibility is invalid", map[string]any{"allowed": []any{"private"}}},
		{"лимит запросов", &errs.RateLimitError{Limit: 5}, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded", map[string]any{"limit": float64(5), "window": "1m"}},
		{"паника", errs.ErrUnexpected, http.StatusInternalServerError, "internal_error", "unexpected server error", nil},
		{"сбой базы", errors.New("connection refused"), http.StatusInternalServerError, "internal_error", "unexpected server error", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			req = req.WithContext(httpx.WithRequestID(req.Context(), "req-1"))
			rec := httptest.NewRecorder()

			httpx.WriteError(rec, req, tc.err)

			require.Equal(t, tc.wantStatus, rec.Code)
			assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))

			var body struct {
				Error struct {
					Code      string `json:"code"`
					Message   string `json:"message"`
					Details   any    `json:"details"`
					RequestID string `json:"requestId"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

			assert.Equal(t, tc.wantCode, body.Error.Code)
			assert.Equal(t, tc.wantMessage, body.Error.Message)
			assert.Equal(t, tc.wantDetails, body.Error.Details)
			assert.Equal(t, "req-1", body.Error.RequestID)
		})
	}
}

// TestDecodeJSON: строгий разбор тела — неизвестные поля, два объекта
// и пустое тело отвергаются.
func TestDecodeJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name    string
		body    string
		wantErr error
	}{
		{name: "один объект", body: `{"name":"x"}`},
		{name: "пустое тело", body: ``, wantErr: errs.ErrRequestBodyRequired},
		{name: "два объекта", body: `{"name":"x"} {}`, wantErr: errs.ErrRequestBodyNotSingleObject},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var dst payload
			err := httpx.DecodeJSON(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body)), &dst, 1024)

			if tc.wantErr == nil {
				require.NoError(t, err)
				assert.Equal(t, "x", dst.Name)

				return
			}

			assert.ErrorIs(t, err, tc.wantErr)
		})
	}

	var dst payload
	err := httpx.DecodeJSON(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"age":1}`)), &dst, 1024)

	var invalid *errs.InvalidJSONError
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, `invalid JSON body: json: unknown field "age"`, invalid.Error())
}

// TestClientIP: X-Forwarded-For (первый адрес) → X-Real-IP → адрес соединения.
func TestClientIP(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = "10.0.0.9:5555"
	assert.Equal(t, "10.0.0.9", httpx.ClientIP(req))

	req.Header.Set("X-Real-IP", " 10.0.0.7 ")
	assert.Equal(t, "10.0.0.7", httpx.ClientIP(req))

	req.Header.Set("X-Forwarded-For", " 203.0.113.7, 10.0.0.1")
	assert.Equal(t, "203.0.113.7", httpx.ClientIP(req))
}
