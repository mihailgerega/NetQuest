package audit

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
)

// AuditRepository — то, что сервису нужно от хранилища аудита.
// Интерфейс объявлен здесь, у потребителя (DIP).
type AuditRepository interface {
	Insert(ctx context.Context, entry model.AuditEntry) error
}
