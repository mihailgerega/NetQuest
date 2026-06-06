package auth

import (
	"context"
	"strings"
	"time"

	"github.com/netquest/netquest/backend/internal/config"
	"github.com/netquest/netquest/backend/internal/security"
	"github.com/netquest/netquest/backend/internal/users"
	"github.com/netquest/netquest/backend/pkg/apperrors"
	"github.com/netquest/netquest/backend/pkg/idgen"
)

type UserStore interface {
	Create(ctx context.Context, user users.CreateUser) (users.User, error)
	FindByEmail(ctx context.Context, email string) (users.User, error)
	FindByID(ctx context.Context, id string) (users.User, error)
	UpsertDemoUser(ctx context.Context, user users.CreateUser) (users.User, error)
}

type RefreshStore interface {
	Create(ctx context.Context, token RefreshToken) error
	FindActiveByHash(ctx context.Context, tokenHash string, now time.Time) (RefreshToken, error)
	RevokeByHash(ctx context.Context, tokenHash string, now time.Time) error
}

type Service struct {
	users   UserStore
	refresh RefreshStore
	jwt     *JWTManager
	cfg     config.SecurityConfig
	now     func() time.Time
}

func NewService(users UserStore, refresh RefreshStore, jwt *JWTManager, cfg config.SecurityConfig) *Service {
	return &Service{users: users, refresh: refresh, jwt: jwt, cfg: cfg, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest, userAgent, ipAddress string) (AuthResponse, error) {
	email := normalizeEmail(req.Email)
	if email == "" {
		return AuthResponse{}, apperrors.Validation("email is required", nil)
	}
	if !strings.Contains(email, "@") {
		return AuthResponse{}, apperrors.Validation("email is invalid", nil)
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = strings.Split(email, "@")[0]
	}

	hash, err := security.HashPassword(req.Password, s.cfg.PasswordHashCost)
	if err != nil {
		return AuthResponse{}, apperrors.Validation(err.Error(), nil)
	}
	id, err := idgen.NewUUID()
	if err != nil {
		return AuthResponse{}, err
	}

	user, err := s.users.Create(ctx, users.CreateUser{
		ID:           id,
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
		Role:         "user",
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return AuthResponse{}, apperrors.Conflict("email is already registered")
		}
		return AuthResponse{}, err
	}
	return s.issueTokens(ctx, user, userAgent, ipAddress)
}

func (s *Service) Login(ctx context.Context, req LoginRequest, userAgent, ipAddress string) (AuthResponse, error) {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(req.Email))
	if err != nil || user.PasswordHash == "" || !security.VerifyPassword(req.Password, user.PasswordHash) {
		return AuthResponse{}, apperrors.Unauthorized("email or password is invalid")
	}
	return s.issueTokens(ctx, user, userAgent, ipAddress)
}

func (s *Service) Demo(ctx context.Context, userAgent, ipAddress string) (AuthResponse, error) {
	if !s.cfg.DemoAuthEnabled {
		return AuthResponse{}, apperrors.Forbidden("demo auth is disabled")
	}
	id := idgen.DeterministicUUID("netquest-demo-user")
	user, err := s.users.UpsertDemoUser(ctx, users.CreateUser{
		ID:          id,
		Email:       "demo@netquest.local",
		DisplayName: "NetQuest Demo",
		Role:        "user",
	})
	if err != nil {
		return AuthResponse{}, err
	}
	return s.issueTokens(ctx, user, userAgent, ipAddress)
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (AuthResponse, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return AuthResponse{}, apperrors.Unauthorized("refresh token is required")
	}
	now := s.now()
	stored, err := s.refresh.FindActiveByHash(ctx, hashToken(refreshToken), now)
	if err != nil {
		return AuthResponse{}, err
	}
	if err := s.refresh.RevokeByHash(ctx, stored.TokenHash, now); err != nil {
		return AuthResponse{}, err
	}
	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil {
		return AuthResponse{}, err
	}
	return s.issueTokens(ctx, user, userAgent, ipAddress)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil
	}
	return s.refresh.RevokeByHash(ctx, hashToken(refreshToken), s.now())
}

func (s *Service) Me(ctx context.Context, userID string) (users.User, error) {
	return s.users.FindByID(ctx, userID)
}

func (s *Service) issueTokens(ctx context.Context, user users.User, userAgent, ipAddress string) (AuthResponse, error) {
	now := s.now()
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Email, user.Role, now)
	if err != nil {
		return AuthResponse{}, err
	}
	refreshToken, err := newOpaqueToken()
	if err != nil {
		return AuthResponse{}, err
	}
	refreshID, err := idgen.NewUUID()
	if err != nil {
		return AuthResponse{}, err
	}
	if err := s.refresh.Create(ctx, RefreshToken{
		ID:        refreshID,
		UserID:    user.ID,
		TokenHash: hashToken(refreshToken),
		UserAgent: userAgent,
		IPAddress: ipAddress,
		ExpiresAt: now.Add(s.cfg.RefreshTokenTTL),
	}); err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.AccessTokenTTL.Seconds()),
	}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
