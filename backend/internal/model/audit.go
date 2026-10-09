package model

import "time"

// AuditEntry — запись журнала аудита: кто, что и над чем сделал.
//
// UserID — указатель: у неудачного входа и logout без access-токена
// пользователя нет, и в audit_logs.user_id пишется NULL.
type AuditEntry struct {
	ID           string
	UserID       *string
	Action       string
	ResourceType string
	ResourceID   string
	IPAddress    string
	UserAgent    string
	Metadata     map[string]any // nil — в базу уйдёт пустой объект {}
	CreatedAt    time.Time
}
