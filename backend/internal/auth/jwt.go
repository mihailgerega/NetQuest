package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/netquest/netquest/backend/internal/config"
)

type JWTManager struct {
	issuer string
	secret []byte
	ttl    time.Duration
}

type AccessClaims struct {
	Email string `json:"email,omitempty"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

type Principal struct {
	UserID string `json:"userId"`
	Email  string `json:"email,omitempty"`
	Role   string `json:"role"`
}

func NewJWTManager(cfg config.SecurityConfig) *JWTManager {
	return &JWTManager{
		issuer: cfg.JWTIssuer,
		secret: []byte(cfg.JWTSecret),
		ttl:    cfg.AccessTokenTTL,
	}
}

func (m *JWTManager) GenerateAccessToken(userID, email, role string, now time.Time) (string, error) {
	claims := AccessClaims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

func (m *JWTManager) ParseAccessToken(raw string) (Principal, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %s", token.Method.Alg())
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer))
	if err != nil {
		return Principal{}, fmt.Errorf("parse access token: %w", err)
	}
	if !token.Valid {
		return Principal{}, fmt.Errorf("invalid access token")
	}
	if claims.Subject == "" {
		return Principal{}, fmt.Errorf("token subject is required")
	}

	return Principal{
		UserID: claims.Subject,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}
