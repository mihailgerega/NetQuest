package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/netquest/netquest/backend/pkg/apperrors"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, user CreateUser) (User, error) {
	if r.db == nil {
		return User{}, apperrors.Internal("postgres is not configured")
	}

	var out User
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (id, email, password_hash, display_name, role, created_at, updated_at)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, now(), now())
		RETURNING id::text, email, COALESCE(display_name, ''), COALESCE(avatar_url, ''), role,
		          created_at, updated_at, deleted_at, COALESCE(password_hash, '')
	`, user.ID, user.Email, user.PasswordHash, user.DisplayName, user.Role).Scan(
		&out.ID,
		&out.Email,
		&out.DisplayName,
		&out.AvatarURL,
		&out.Role,
		&out.CreatedAt,
		&out.UpdatedAt,
		&out.DeletedAt,
		&out.PasswordHash,
	)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return out, nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (User, error) {
	if r.db == nil {
		return User{}, apperrors.Internal("postgres is not configured")
	}

	var user User
	err := r.db.QueryRow(ctx, `
		SELECT id::text, email, COALESCE(display_name, ''), COALESCE(avatar_url, ''), role,
		       created_at, updated_at, deleted_at, COALESCE(password_hash, '')
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
		&user.PasswordHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, apperrors.NotFound("user not found")
	}
	if err != nil {
		return User{}, fmt.Errorf("find user by email: %w", err)
	}

	return user, nil
}

func (r *Repository) UpsertDemoUser(ctx context.Context, user CreateUser) (User, error) {
	if r.db == nil {
		return User{}, apperrors.Internal("postgres is not configured")
	}

	var out User
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (id, email, password_hash, display_name, role, created_at, updated_at)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, now(), now())
		ON CONFLICT (email) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    role = EXCLUDED.role,
		    updated_at = now(),
		    deleted_at = NULL
		RETURNING id::text, email, COALESCE(display_name, ''), COALESCE(avatar_url, ''), role,
		          created_at, updated_at, deleted_at, COALESCE(password_hash, '')
	`, user.ID, user.Email, user.PasswordHash, user.DisplayName, user.Role).Scan(
		&out.ID,
		&out.Email,
		&out.DisplayName,
		&out.AvatarURL,
		&out.Role,
		&out.CreatedAt,
		&out.UpdatedAt,
		&out.DeletedAt,
		&out.PasswordHash,
	)
	if err != nil {
		return User{}, fmt.Errorf("upsert demo user: %w", err)
	}
	return out, nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (User, error) {
	if r.db == nil {
		return User{}, apperrors.Internal("postgres is not configured")
	}

	var user User
	err := r.db.QueryRow(ctx, `
		SELECT id::text, email, COALESCE(display_name, ''), COALESCE(avatar_url, ''), role,
		       created_at, updated_at, deleted_at, COALESCE(password_hash, '')
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
		&user.PasswordHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, apperrors.NotFound("user not found")
	}
	if err != nil {
		return User{}, fmt.Errorf("find user by id: %w", err)
	}

	return user, nil
}
