package audit

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/netquest/netquest/backend/pkg/apperrors"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Insert(ctx context.Context, entry Entry) error {
	if r.db == nil {
		return apperrors.Internal("postgres is not configured")
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, action, resource_type, resource_id, ip_address, user_agent, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::inet, $7, COALESCE($8, '{}'::jsonb), $9)
	`, entry.ID, entry.UserID, entry.Action, entry.ResourceType, entry.ResourceID, entry.IPAddress, entry.UserAgent, entry.Metadata, entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return nil
}
