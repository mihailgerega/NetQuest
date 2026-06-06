package auth

import (
	"testing"
	"time"

	"github.com/netquest/netquest/backend/internal/config"
	"github.com/netquest/netquest/backend/internal/security"
)

func TestJWTManagerGenerateAndParse(t *testing.T) {
	manager := NewJWTManager(config.SecurityConfig{
		JWTIssuer:      "netquest-test",
		JWTSecret:      "test-secret-with-at-least-thirty-two-chars",
		AccessTokenTTL: time.Minute,
	})

	token, err := manager.GenerateAccessToken("user-1", "user@example.com", "user", time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	principal, err := manager.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken returned error: %v", err)
	}
	if principal.UserID != "user-1" || principal.Email != "user@example.com" || principal.Role != "user" {
		t.Fatalf("unexpected principal: %#v", principal)
	}
}

func TestJWTManagerRejectsWrongSecret(t *testing.T) {
	first := NewJWTManager(config.SecurityConfig{JWTIssuer: "netquest-test", JWTSecret: "first-secret-with-at-least-thirty-two-chars", AccessTokenTTL: time.Minute})
	second := NewJWTManager(config.SecurityConfig{JWTIssuer: "netquest-test", JWTSecret: "second-secret-with-at-least-thirty-two-chars", AccessTokenTTL: time.Minute})

	token, err := first.GenerateAccessToken("user-1", "user@example.com", "user", time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}
	if _, err := second.ParseAccessToken(token); err == nil {
		t.Fatal("expected wrong secret to be rejected")
	}
}

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := security.HashPassword("very-secure-password", 10)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if !security.VerifyPassword("very-secure-password", hash) {
		t.Fatal("expected password to verify")
	}
	if security.VerifyPassword("wrong-password", hash) {
		t.Fatal("expected wrong password to fail")
	}
}
