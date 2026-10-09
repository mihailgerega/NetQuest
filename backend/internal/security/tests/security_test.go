package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netquest/netquest/backend/internal/security"
)

const (
	issuer = "netquest-test"
	secret = "test-secret-with-at-least-thirty-two-chars"
)

// TestJWTRoundTrip: выпущенный токен разбирается обратно в того же пользователя.
func TestJWTRoundTrip(t *testing.T) {
	t.Parallel()

	manager := security.NewJWTManager(issuer, secret, time.Minute)

	token, err := manager.GenerateAccessToken("user-1", "user@example.com", "user", time.Now().UTC())
	require.NoError(t, err)

	principal, err := manager.ParseAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", principal.UserID)
	assert.Equal(t, "user@example.com", principal.Email)
	assert.Equal(t, "user", principal.Role)
}

// TestJWTRejects: чужой секрет, чужой издатель, истёкший срок, alg none
// и токен без sub не проходят проверку.
func TestJWTRejects(t *testing.T) {
	t.Parallel()

	manager := security.NewJWTManager(issuer, secret, time.Minute)
	now := time.Now().UTC()

	otherSecret, err := security.NewJWTManager(issuer, "another-secret-with-at-least-thirty-two-chars", time.Minute).
		GenerateAccessToken("user-1", "", "user", now)
	require.NoError(t, err)

	otherIssuer, err := security.NewJWTManager("someone-else", secret, time.Minute).
		GenerateAccessToken("user-1", "", "user", now)
	require.NoError(t, err)

	expired, err := manager.GenerateAccessToken("user-1", "", "user", now.Add(-time.Hour))
	require.NoError(t, err)

	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "user-1", "iss": issuer}).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	noSubject, err := manager.GenerateAccessToken("", "", "user", now)
	require.NoError(t, err)

	for name, token := range map[string]string{
		"чужой секрет":   otherSecret,
		"чужой издатель": otherIssuer,
		"истёк":          expired,
		"alg none":       none,
		"без sub":        noSubject,
		"мусор":          "not-a-jwt",
	} {
		_, err := manager.ParseAccessToken(token)
		assert.Error(t, err, name)
	}
}

// TestPassword: хеш проверяется; короткий и слишком длинный пароль отвергаются
// с текстом, который уйдёт клиенту.
func TestPassword(t *testing.T) {
	t.Parallel()

	hash, err := security.HashPassword("very-secure-password", 10)
	require.NoError(t, err)
	assert.True(t, security.VerifyPassword("very-secure-password", hash))
	assert.False(t, security.VerifyPassword("wrong-password", hash))
	assert.False(t, security.VerifyPassword("very-secure-password", "not-a-bcrypt-hash"))

	_, err = security.HashPassword("short", 10)
	require.EqualError(t, err, "password must be at least 12 characters")

	_, err = security.HashPassword(strings.Repeat("x", 73), 10)
	require.EqualError(t, err, "hash password: bcrypt: password length exceeds 72 bytes")
}
