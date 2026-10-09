// Package audit — сервисный слой журнала аудита: кто, что и когда сделал
// (вход, создание проекта, запуск симуляции, прохождение квеста...).
//
// Место в цепочке: API-хендлеры (через api/auditlog) → service/application/audit
// → repository/audit. Ошибка записи аудита не проваливает запрос пользователя —
// хендлер её только логирует.
package audit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/netquest/netquest/backend/internal/model"
)

// errActionRequired — запись без действия бессмысленна.
var errActionRequired = errors.New("действие аудита не задано")

// service — сервис аудита.
type service struct {
	auditRepository AuditRepository
}

// New создаёт сервис аудита.
func New(auditRepository AuditRepository) *service {
	return &service{
		auditRepository: auditRepository,
	}
}

// Record дописывает запись в журнал. ID и время ставятся здесь, если их нет.
func (s *service) Record(ctx context.Context, entry model.AuditEntry) error {
	if entry.Action == "" {
		return errActionRequired
	}

	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}

	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}

	if err := s.auditRepository.Insert(ctx, entry); err != nil {
		return fmt.Errorf("сохранить запись аудита: %w", err)
	}

	return nil
}
