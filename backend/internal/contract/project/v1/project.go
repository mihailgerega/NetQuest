// Package projectv1 — HTTP-контракт проектов: /api/v1/projects/*.
//
// Имена типов запросов значимы: encoding/json вставляет их в текст ошибки
// разбора, который уходит клиенту в ответе 400.
package projectv1

import "time"

// CreateRequest — тело POST /api/v1/projects.
type CreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

// UpdateRequest — тело PATCH /api/v1/projects/{projectId}: отсутствующее
// поле (nil) не меняется.
type UpdateRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Visibility  *string `json:"visibility"`
}

// Project — проект в ответах.
type Project struct {
	ID          string     `json:"id"`
	OwnerID     string     `json:"ownerId"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Visibility  string     `json:"visibility"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

// ProjectResponse — ответ с одним проектом.
type ProjectResponse struct {
	Project Project `json:"project"`
}

// ListResponse — ответ GET /api/v1/projects.
type ListResponse struct {
	Projects []Project `json:"projects"`
}
