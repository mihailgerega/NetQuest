package tests

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netquest/netquest/backend/internal/auth"
	"github.com/netquest/netquest/backend/internal/httpx"
	"github.com/netquest/netquest/backend/internal/middleware"
	"github.com/netquest/netquest/backend/internal/middleware/mocks"
	"github.com/netquest/netquest/backend/internal/model"
)

// okHandler отвечает 200 и запоминает, что до него дошли.
type okHandler struct{ called bool }

func (h *okHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.called = true
	w.WriteHeader(http.StatusOK)
}

// TestRequireAuth: без заголовка, с чужой схемой и с невалидным токеном —
// 401 до обработчика; с валидным — пользователь в ctx.
func TestRequireAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		header      string
		parseErr    error
		wantParsed  string // какой токен дойдёт до парсера; пусто — не дойдёт
		wantStatus  int
		wantMessage string
	}{
		{name: "нет заголовка", wantStatus: http.StatusUnauthorized, wantMessage: "authorization bearer token is required"},
		{name: "схема Basic", header: "Basic abc", wantStatus: http.StatusUnauthorized, wantMessage: "authorization bearer token is invalid"},
		{name: "bearer строчными", header: "bearer abc", wantStatus: http.StatusUnauthorized, wantMessage: "authorization bearer token is invalid"},
		// После обрезки пробелов остаётся "Bearer" без пробела — это уже не схема Bearer.
		{name: "пустой токен", header: "Bearer    ", wantStatus: http.StatusUnauthorized, wantMessage: "authorization bearer token is invalid"},
		{name: "токен не прошёл проверку", header: "Bearer bad", parseErr: errors.New("подпись"), wantParsed: "bad", wantStatus: http.StatusUnauthorized, wantMessage: "authorization bearer token is invalid"},
		{name: "валидный токен", header: "  Bearer  good  ", wantParsed: "good", wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser := mocks.NewAccessTokenParser(t)
			if tc.wantParsed != "" {
				parser.EXPECT().ParseAccessToken(tc.wantParsed).Return(model.Principal{UserID: "user-1"}, tc.parseErr).Once()
			}

			var principal model.Principal

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				principal, _ = auth.PrincipalFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", http.NoBody)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}

			rec := httptest.NewRecorder()
			middleware.RequireAuth(parser)(next).ServeHTTP(rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantMessage != "" {
				assert.Contains(t, rec.Body.String(), tc.wantMessage)
				return
			}

			assert.Equal(t, "user-1", principal.UserID)
		})
	}
}

// TestCORS: разрешённый origin получает CORS-заголовки, чужой — нет;
// preflight отвечает 204 и не доходит до обработчика.
func TestCORS(t *testing.T) {
	t.Parallel()

	cors := middleware.CORS(middleware.CORSConfig{AllowedOrigins: []string{"http://localhost:3000"}, AllowCredentials: true})

	next := &okHandler{}
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	cors(next).ServeHTTP(rec, req)

	assert.True(t, next.called)
	assert.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "GET,POST,PATCH,DELETE,OPTIONS", rec.Header().Get("Access-Control-Allow-Methods"))

	next = &okHandler{}
	preflight := httptest.NewRequest(http.MethodOptions, "/", http.NoBody)
	preflight.Header.Set("Origin", "http://evil.example")
	rec = httptest.NewRecorder()
	cors(next).ServeHTTP(rec, preflight)

	assert.False(t, next.called)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

// TestRequestID: свой ID клиента сохраняется, без него — генерируется,
// и в обоих случаях он есть и в ответе, и в ctx.
func TestRequestID(t *testing.T) {
	t.Parallel()

	var fromContext string

	handler := middleware.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		fromContext = httpx.RequestID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Request-ID", "client-id")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "client-id", rec.Header().Get("X-Request-ID"))
	assert.Equal(t, "client-id", fromContext)

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	assert.Len(t, rec.Header().Get("X-Request-ID"), 36, "UUID")
	assert.Equal(t, rec.Header().Get("X-Request-ID"), fromContext)
}

// TestRecover: паника обработчика — 500 в формате ошибок API, а не обрыв соединения.
func TestRecover(t *testing.T) {
	t.Parallel()

	handler := middleware.Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "unexpected server error")
}

// TestChainOrder: первый middleware в списке — внешний.
func TestChainOrder(t *testing.T) {
	t.Parallel()

	var order []string

	mark := func(name string) middleware.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}

	handler := middleware.Chain(&okHandler{}, mark("a"), mark("b"), mark("c"))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	assert.Equal(t, []string{"a", "b", "c"}, order)
}
