// Package audit — репозиторий журнала аудита поверх PostgreSQL.
//
// Место в цепочке: service/application/audit → repository/audit → таблица audit_logs.
// Журнал только пополняется: читает его не API, а администратор напрямую в базе.
package audit

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/netquest/netquest/backend/internal/model"
)

// repository пишет записи аудита через общий пул соединений.
type repository struct {
	pool *pgxpool.Pool
}

// New создаёт репозиторий аудита поверх пула.
func New(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}

// Insert добавляет запись в журнал. Пустой IP пишется как NULL; metadata без
// значения (nil map → NULL) заменяется пустым объектом {}.
func (r *repository) Insert(ctx context.Context, entry model.AuditEntry) error {
	const query = `
		INSERT INTO audit_logs (id, user_id, action, resource_type, resource_id, ip_address, user_agent, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::inet, $7, COALESCE($8, '{}'::jsonb), $9)`

	if _, err := r.pool.Exec(ctx, query,
		entry.ID, entry.UserID, entry.Action, entry.ResourceType, entry.ResourceID,
		entry.IPAddress, entry.UserAgent, entry.Metadata, entry.CreatedAt); err != nil {
		return fmt.Errorf("записать аудит: %w", err)
	}

	return nil
}
