// Package topologyv1 — HTTP-контракт версий топологии:
// /api/v1/projects/{projectId}/topologies и /api/v1/topologies/{topologyId}.
//
// Имена типов запросов значимы: encoding/json вставляет их в текст ошибки
// разбора, который уходит клиенту в ответе 400.
package topologyv1

import (
	"encoding/json"
	"time"

	"github.com/netquest/netquest/backend/internal/model"
)

// CreateRequest — тело POST /api/v1/projects/{projectId}/topologies.
// Data — документ топологии; разбирает и проверяет его валидатор.
type CreateRequest struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

// Topology — версия топологии в ответах. Data отдаётся как сохранён.
type Topology struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"projectId"`
	Version   int             `json:"version"`
	Name      string          `json:"name"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
	DeletedAt *time.Time      `json:"deletedAt,omitempty"`
	CreatedBy *string         `json:"createdBy,omitempty"`
}

// ListResponse — ответ со списком версий.
type ListResponse struct {
	Topologies []Topology `json:"topologies"`
}

// TopologyResponse — ответ с одной версией.
type TopologyResponse struct {
	Topology Topology `json:"topology"`
}

// CreateResponse — ответ сохранения: версия и результат её валидации.
type CreateResponse struct {
	Topology   Topology               `json:"topology"`
	Validation model.ValidationResult `json:"validation"`
}
