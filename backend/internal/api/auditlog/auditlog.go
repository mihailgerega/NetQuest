// Package auditlog — запись аудита из HTTP-хендлеров.
//
// Хендлеры зовут Record после успешного (или, для входа, неудачного) действия.
// Аудит не должен ломать запрос пользователя: ошибка записи только логируется,
// а ответ уходит как обычно.
package auditlog

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/netquest/netquest/backend/internal/httpx"
	"github.com/netquest/netquest/backend/internal/model"
)

// Recorder — то, что хендлерам нужно от сервиса аудита.
type Recorder interface {
	Record(ctx context.Context, entry model.AuditEntry) error
}

// Record пишет запись аудита о действии над ресурсом. IP и User-Agent
// берутся из запроса. userID — nil, если пользователь неизвестен (неудачный вход).
// recorder — nil в тестах хендлеров, которым аудит не важен.
func Record(r *http.Request, recorder Recorder, userID *string, action, resourceType, resourceID string) {
	if recorder == nil {
		return
	}

	err := recorder.Record(r.Context(), model.AuditEntry{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		IPAddress:    httpx.ClientIP(r),
		UserAgent:    r.UserAgent(),
	})
	if err != nil {
		slog.WarnContext(r.Context(), "не удалось записать аудит", "action", action, "error", err)
	}
}
