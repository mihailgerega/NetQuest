package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/netquest/netquest/backend/pkg/apperrors"
)

type RefreshRepository struct {
	db *pgxpool.Pool
}

func NewRefreshRepository(db *pgxpool.Pool) *RefreshRepository {
	return &RefreshRepository{db: db}
}

func (r *RefreshRepository) Create(ctx context.Context, token RefreshToken) error {
	if r.db == nil {
		return apperrors.Internal("postgres is not configured")
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, user_agent, ip_address, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, '')::inet, $6, now(), now())
	`, token.ID, token.UserID, token.TokenHash, token.UserAgent, token.IPAddress, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *RefreshRepository) FindActiveByHash(ctx context.Context, tokenHash string, now time.Time) (RefreshToken, error) {
	if r.db == nil {
		return RefreshToken{}, apperrors.Internal("postgres is not configured")
	}

	var token RefreshToken
	err := r.db.QueryRow(ctx, `
		SELECT id::text, user_id::text, token_hash, COALESCE(user_agent, ''), COALESCE(ip_address::text, ''),
		       expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2
	`, tokenHash, now).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.UserAgent,
		&token.IPAddress,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return RefreshToken{}, apperrors.Unauthorized("refresh token is invalid")
	}
	if err != nil {
		return RefreshToken{}, fmt.Errorf("find refresh token: %w", err)
	}
	return token, nil
}

func (r *RefreshRepository) RevokeByHash(ctx context.Context, tokenHash string, now time.Time) error {
	if r.db == nil {
		return apperrors.Internal("postgres is not configured")
	}

	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = COALESCE(revoked_at, $2), updated_at = now()
		WHERE token_hash = $1
	`, tokenHash, now)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}
