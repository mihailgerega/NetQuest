package auth

import (
	"time"

	"github.com/netquest/netquest/backend/internal/users"
)

const RefreshTokenCookieName = "netquest_refresh_token"

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type AuthResponse struct {
	User         users.User `json:"user"`
	AccessToken  string     `json:"accessToken"`
	RefreshToken string     `json:"refreshToken,omitempty"`
	TokenType    string     `json:"tokenType"`
	ExpiresIn    int64      `json:"expiresIn"`
}

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	UserAgent string
	IPAddress string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
