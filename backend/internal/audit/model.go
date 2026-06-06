package audit

import "time"

type Entry struct {
	ID           string         `json:"id"`
	UserID       *string        `json:"userId,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resourceType,omitempty"`
	ResourceID   string         `json:"resourceId,omitempty"`
	IPAddress    string         `json:"ipAddress,omitempty"`
	UserAgent    string         `json:"userAgent,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}
