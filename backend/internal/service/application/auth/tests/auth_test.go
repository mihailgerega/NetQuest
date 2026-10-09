package tests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/security"
	authService "github.com/netquest/netquest/backend/internal/service/application/auth"
	"github.com/netquest/netquest/backend/internal/service/application/auth/mocks"
	"github.com/netquest/netquest/backend/internal/service/input"
	"github.com/netquest/netquest/backend/pkg/idgen"
)

// Unit-тесты сервиса аутентификации. Репозитории — моки, JWT — настоящий
// (security.JWTManager): так проверяется и то, что выданный токен разбирается.
// bcrypt настоящий с минимальной для конфига сложностью.

const (
	testPassword = "correct horse battery"
	testCost     = 10
)

var testClient = input.ClientInfo{UserAgent: "test-agent", IPAddress: "127.0.0.1"}

// authServiceAPI — методы сервиса глазами теста (тип сервиса неэкспортируемый).
type authServiceAPI interface {
	Register(ctx context.Context, in input.RegisterInput, client input.ClientInfo) (model.AuthTokens, error)
	Login(ctx context.Context, in input.LoginInput, client input.ClientInfo) (model.AuthTokens, error)
	Demo(ctx context.Context, client input.ClientInfo) (model.AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string, client input.ClientInfo) (model.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
}

func newService(t *testing.T, demoEnabled bool) (authServiceAPI, *mocks.UserRepository, *mocks.RefreshTokenRepository, *security.JWTManager) {
	t.Helper()

	users := mocks.NewUserRepository(t)
	refresh := mocks.NewRefreshTokenRepository(t)
	jwtManager := security.NewJWTManager("netquest-test", "test-secret-with-at-least-thirty-two-chars", time.Minute)

	svc := authService.New(users, refresh, jwtManager, authService.Settings{
		AccessTokenTTL:   time.Minute,
		RefreshTokenTTL:  time.Hour,
		PasswordHashCost: testCost,
		DemoAuthEnabled:  demoEnabled,
	})

	return svc, users, refresh, jwtManager
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// TestRegister: email нормализуется, имя берётся из email, пароль хешируется,
// в базу уходит хеш refresh-токена, а клиенту — рабочий JWT и сам токен.
func TestRegister(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, users, refresh, jwtManager := newService(t, true)

	users.EXPECT().Create(ctx, mock.Anything).
		RunAndReturn(func(_ context.Context, user model.User) (model.User, error) {
			assert.Equal(t, "alice@example.com", user.Email)
			assert.Equal(t, "alice", user.DisplayName)
			assert.Equal(t, model.RoleUser, user.Role)
			assert.True(t, security.VerifyPassword(testPassword, user.PasswordHash))

			return user, nil
		}).Once()

	var stored model.RefreshToken
	refresh.EXPECT().Create(ctx, mock.Anything).
		Run(func(_ context.Context, token model.RefreshToken) { stored = token }).
		Return(nil).Once()

	tokens, err := svc.Register(ctx, input.RegisterInput{Email: "  Alice@Example.COM ", Password: testPassword}, testClient)
	require.NoError(t, err)

	assert.Equal(t, model.TokenTypeBearer, tokens.TokenType)
	assert.Equal(t, int64(60), tokens.ExpiresIn)
	assert.Equal(t, sha256Hex(tokens.RefreshToken), stored.TokenHash, "в базе — хеш, а не сам токен")
	assert.Equal(t, testClient.UserAgent, stored.UserAgent)
	assert.Equal(t, testClient.IPAddress, stored.IPAddress)

	principal, err := jwtManager.ParseAccessToken(tokens.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, tokens.User.ID, principal.UserID)
	assert.Equal(t, "alice@example.com", principal.Email)
}

// TestRegisterValidation: ошибки клиента — 422 с точным текстом, база не трогается.
func TestRegisterValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      input.RegisterInput
		wantErr string
	}{
		{name: "нет email", in: input.RegisterInput{Password: testPassword}, wantErr: "email is required"},
		{name: "email без @", in: input.RegisterInput{Email: "alice.example.com", Password: testPassword}, wantErr: "email is invalid"},
		{name: "короткий пароль", in: input.RegisterInput{Email: "a@b.c", Password: "short"}, wantErr: "password must be at least 12 characters"},
		{name: "пароль длиннее 72 байт", in: input.RegisterInput{Email: "a@b.c", Password: strings.Repeat("x", 80)}, wantErr: "hash password: bcrypt: password length exceeds 72 bytes"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, _, _, _ := newService(t, true)

			_, err := svc.Register(context.Background(), tc.in, testClient)

			var validationErr *errs.ValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, tc.wantErr, validationErr.Message)
		})
	}
}

// TestRegisterEmailTaken: занятый email отдаётся без обёртки — это 409.
func TestRegisterEmailTaken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, users, _, _ := newService(t, true)

	users.EXPECT().Create(ctx, mock.Anything).Return(model.User{}, errs.ErrEmailTaken).Once()

	_, err := svc.Register(ctx, input.RegisterInput{Email: "a@b.c", Password: testPassword}, testClient)

	assert.ErrorIs(t, err, errs.ErrEmailTaken)
}

// TestLoginFailures: нет пользователя, нет пароля (demo), не тот пароль и даже
// сбой базы — одна и та же ErrInvalidCredentials.
func TestLoginFailures(t *testing.T) {
	t.Parallel()

	hash, err := security.HashPassword(testPassword, testCost)
	require.NoError(t, err)

	tests := []struct {
		name     string
		user     model.User
		findErr  error
		password string
	}{
		{name: "нет пользователя", findErr: errs.ErrUserNotFound, password: testPassword},
		{name: "сбой базы", findErr: errors.New("PostgreSQL недоступен"), password: testPassword},
		{name: "demo без пароля", user: model.User{ID: "demo"}, password: testPassword},
		{name: "не тот пароль", user: model.User{ID: "u1", PasswordHash: hash}, password: "wrong password!"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			svc, users, _, _ := newService(t, true)

			users.EXPECT().FindByEmail(ctx, "alice@example.com").Return(tc.user, tc.findErr).Once()

			_, err := svc.Login(ctx, input.LoginInput{Email: " ALICE@example.com", Password: tc.password}, testClient)

			assert.ErrorIs(t, err, errs.ErrInvalidCredentials)
		})
	}
}

// TestRefreshRotation: старый токен отзывается, выдаётся новый.
func TestRefreshRotation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, users, refresh, _ := newService(t, true)

	oldHash := sha256Hex("old-token")

	refresh.EXPECT().FindActiveByHash(ctx, oldHash, mock.Anything).
		Return(model.RefreshToken{UserID: "user-1", TokenHash: oldHash}, nil).Once()
	refresh.EXPECT().RevokeByHash(ctx, oldHash, mock.Anything).Return(nil).Once()
	users.EXPECT().FindByID(ctx, "user-1").Return(model.User{ID: "user-1"}, nil).Once()
	refresh.EXPECT().Create(ctx, mock.MatchedBy(func(token model.RefreshToken) bool {
		return token.UserID == "user-1" && token.TokenHash != oldHash
	})).Return(nil).Once()

	tokens, err := svc.Refresh(ctx, "  old-token ", testClient)

	require.NoError(t, err)
	assert.NotEqual(t, "old-token", tokens.RefreshToken)
}

// TestRefreshAndLogoutWithoutToken: refresh без токена — 401, logout без токена — ничего.
func TestRefreshAndLogoutWithoutToken(t *testing.T) {
	t.Parallel()

	svc, _, _, _ := newService(t, true)

	_, err := svc.Refresh(context.Background(), "   ", testClient)
	assert.ErrorIs(t, err, errs.ErrRefreshTokenRequired)

	assert.NoError(t, svc.Logout(context.Background(), ""))
}

// TestDemo: выключенный demo-вход — 403; включённый — общий аккаунт
// с детерминированным ID.
func TestDemo(t *testing.T) {
	t.Parallel()

	disabled, _, _, _ := newService(t, false)
	_, err := disabled.Demo(context.Background(), testClient)
	assert.ErrorIs(t, err, errs.ErrDemoAuthDisabled)

	ctx := context.Background()
	svc, users, refresh, _ := newService(t, true)

	users.EXPECT().UpsertDemoUser(ctx, mock.MatchedBy(func(user model.User) bool {
		return user.Email == "demo@netquest.local" && user.ID == idgen.DeterministicUUID("netquest-demo-user")
	})).RunAndReturn(func(_ context.Context, user model.User) (model.User, error) { return user, nil }).Once()
	refresh.EXPECT().Create(ctx, mock.Anything).Return(nil).Once()

	tokens, err := svc.Demo(ctx, testClient)
	require.NoError(t, err)
	assert.Equal(t, "NetQuest Demo", tokens.User.DisplayName)
}
