package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/netquest/netquest/backend/internal/config"
	"github.com/netquest/netquest/backend/internal/users"
)

func TestRegisterSetsRefreshCookieAndOmitsRefreshTokenFromBody(t *testing.T) {
	cfg := config.SecurityConfig{
		JWTIssuer:        "netquest-test",
		JWTSecret:        "test-secret",
		AccessTokenTTL:   time.Minute,
		RefreshTokenTTL:  time.Hour,
		PasswordHashCost: 4,
	}
	service := NewService(newFakeUserStore(), newFakeRefreshStore(), NewJWTManager(cfg), cfg)
	handler := NewHandler(service, nil, nil, 1<<20, false)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"demo@example.com","password":"correct horse battery staple","displayName":"Demo"}`))
	rec := httptest.NewRecorder()
	handler.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var body AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode auth response: %v", err)
	}
	if body.AccessToken == "" {
		t.Fatalf("expected access token in response")
	}
	if body.RefreshToken != "" {
		t.Fatalf("refresh token must not be returned in JSON body")
	}
	cookie := rec.Result().Cookies()
	if len(cookie) == 0 || cookie[0].Name != RefreshTokenCookieName || cookie[0].Value == "" || !cookie[0].HttpOnly {
		t.Fatalf("expected httpOnly refresh cookie, got %#v", cookie)
	}
}

type fakeUserStore struct {
	byID    map[string]users.User
	byEmail map[string]users.User
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{byID: map[string]users.User{}, byEmail: map[string]users.User{}}
}

func (s *fakeUserStore) Create(_ context.Context, input users.CreateUser) (users.User, error) {
	if _, exists := s.byEmail[input.Email]; exists {
		return users.User{}, errors.New("duplicate email")
	}
	now := time.Now().UTC()
	user := users.User{
		ID:           input.ID,
		Email:        input.Email,
		DisplayName:  input.DisplayName,
		Role:         input.Role,
		PasswordHash: input.PasswordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.byID[user.ID] = user
	s.byEmail[user.Email] = user
	return user, nil
}

func (s *fakeUserStore) FindByEmail(_ context.Context, email string) (users.User, error) {
	user, ok := s.byEmail[email]
	if !ok {
		return users.User{}, errors.New("not found")
	}
	return user, nil
}

func (s *fakeUserStore) FindByID(_ context.Context, id string) (users.User, error) {
	user, ok := s.byID[id]
	if !ok {
		return users.User{}, errors.New("not found")
	}
	return user, nil
}

func (s *fakeUserStore) UpsertDemoUser(_ context.Context, input users.CreateUser) (users.User, error) {
	now := time.Now().UTC()
	user := users.User{ID: input.ID, Email: input.Email, DisplayName: input.DisplayName, Role: input.Role, CreatedAt: now, UpdatedAt: now}
	s.byID[user.ID] = user
	s.byEmail[user.Email] = user
	return user, nil
}

type fakeRefreshStore struct {
	tokens map[string]RefreshToken
}

func newFakeRefreshStore() *fakeRefreshStore {
	return &fakeRefreshStore{tokens: map[string]RefreshToken{}}
}

func (s *fakeRefreshStore) Create(_ context.Context, token RefreshToken) error {
	s.tokens[token.TokenHash] = token
	return nil
}

func (s *fakeRefreshStore) FindActiveByHash(_ context.Context, tokenHash string, now time.Time) (RefreshToken, error) {
	token, ok := s.tokens[tokenHash]
	if !ok || token.RevokedAt != nil || token.ExpiresAt.Before(now) {
		return RefreshToken{}, errors.New("refresh token not found")
	}
	return token, nil
}

func (s *fakeRefreshStore) RevokeByHash(_ context.Context, tokenHash string, now time.Time) error {
	token := s.tokens[tokenHash]
	token.RevokedAt = &now
	s.tokens[tokenHash] = token
	return nil
}
