package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	auditMocks "github.com/netquest/netquest/backend/internal/api/auditlog/mocks"
	authAPI "github.com/netquest/netquest/backend/internal/api/auth/v1"
	"github.com/netquest/netquest/backend/internal/api/auth/v1/mocks"
	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// Тесты HTTP-обработчиков аутентификации: сервис — мок, проверяется то, что
// делает сам API-слой — разбор тела, refresh-cookie, ответ без refresh-токена,
// перевод ошибок в коды и записи аудита.

const jsonLimit = 1 << 20

var tokens = model.AuthTokens{
	User:         model.User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo", Role: model.RoleUser},
	AccessToken:  "access-token",
	RefreshToken: "refresh-token",
	TokenType:    model.TokenTypeBearer,
	ExpiresIn:    60,
}

// TestRegisterSetsRefreshCookie: 201, access-токен в теле, refresh-токен —
// только в httpOnly-cookie, регистрация попадает в аудит.
func TestRegisterSetsRefreshCookie(t *testing.T) {
	t.Parallel()

	service := mocks.NewAuthService(t)
	recorder := auditMocks.NewRecorder(t)
	api := authAPI.New(service, recorder, jsonLimit, false)

	service.EXPECT().Register(mock.Anything, input.RegisterInput{Email: "demo@example.com", Password: "correct horse battery staple", DisplayName: "Demo"}, mock.Anything).
		Return(tokens, nil).Once()
	recorder.EXPECT().Record(mock.Anything, mock.MatchedBy(func(entry model.AuditEntry) bool {
		return entry.Action == "register" && entry.ResourceID == "user-1" && *entry.UserID == "user-1"
	})).Return(nil).Once()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
		strings.NewReader(`{"email":"demo@example.com","password":"correct horse battery staple","displayName":"Demo"}`))
	rec := httptest.NewRecorder()

	api.Register(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "access-token", body["accessToken"])
	assert.NotContains(t, body, "refreshToken", "refresh-токен не должен попадать в тело")

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, authAPI.RefreshTokenCookieName, cookies[0].Name)
	assert.Equal(t, "refresh-token", cookies[0].Value)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
}

// TestRegisterBadJSON: кривое тело — 400 до вызова сервиса.
func TestRegisterBadJSON(t *testing.T) {
	t.Parallel()

	api := authAPI.New(mocks.NewAuthService(t), nil, jsonLimit, false)

	rec := httptest.NewRecorder()
	api.Register(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":`)))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":"bad_request"`)
}

// TestLoginFailureIsAudited: неудачный вход — 401 и запись failed_login без пользователя.
func TestLoginFailureIsAudited(t *testing.T) {
	t.Parallel()

	service := mocks.NewAuthService(t)
	recorder := auditMocks.NewRecorder(t)
	api := authAPI.New(service, recorder, jsonLimit, false)

	service.EXPECT().Login(mock.Anything, mock.Anything, mock.Anything).Return(model.AuthTokens{}, errs.ErrInvalidCredentials).Once()
	recorder.EXPECT().Record(mock.Anything, mock.MatchedBy(func(entry model.AuditEntry) bool {
		return entry.Action == "failed_login" && entry.UserID == nil
	})).Return(nil).Once()

	rec := httptest.NewRecorder()
	api.Login(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"a@b.c","password":"x"}`)))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "email or password is invalid")
}

// TestRefreshReadsCookie: без тела refresh-токен берётся из cookie.
func TestRefreshReadsCookie(t *testing.T) {
	t.Parallel()

	service := mocks.NewAuthService(t)
	api := authAPI.New(service, nil, jsonLimit, true)

	service.EXPECT().Refresh(mock.Anything, "from-cookie", mock.Anything).Return(tokens, nil).Once()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", http.NoBody)
	req.AddCookie(&http.Cookie{Name: authAPI.RefreshTokenCookieName, Value: "from-cookie"})
	rec := httptest.NewRecorder()

	api.Refresh(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.True(t, cookies[0].Secure, "SECURE_COOKIE=true ставит флаг Secure")
}

// TestLogoutClearsCookie: logout отзывает токен из тела и стирает cookie.
func TestLogoutClearsCookie(t *testing.T) {
	t.Parallel()

	service := mocks.NewAuthService(t)
	api := authAPI.New(service, nil, jsonLimit, false)

	service.EXPECT().Logout(mock.Anything, "from-body").Return(nil).Once()

	rec := httptest.NewRecorder()
	api.Logout(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", strings.NewReader(`{"refreshToken":"from-body"}`)))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"ok":true}`, rec.Body.String())

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, -1, cookies[0].MaxAge)
}

// TestMeWithoutPrincipal: без пользователя в ctx (хендлер вызван мимо RequireAuth) — 401.
func TestMeWithoutPrincipal(t *testing.T) {
	t.Parallel()

	api := authAPI.New(mocks.NewAuthService(t), nil, jsonLimit, false)

	rec := httptest.NewRecorder()
	api.Me(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", http.NoBody))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "authentication is required")
}
