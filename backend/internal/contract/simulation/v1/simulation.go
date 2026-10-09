// Package simulationv1 — HTTP-контракт симуляций: /api/v1/simulations/*
// и кадры WebSocket /api/v1/ws.
//
// Имена типов запросов значимы: encoding/json вставляет их в текст ошибки
// разбора ("Go struct field Scenario.scenario.type"), который уходит клиенту в ответе 400.
package simulationv1

import (
	"encoding/json"
	"time"

	"github.com/netquest/netquest/backend/internal/model"
)

// Scenario — сценарий симуляции в запросе.
type Scenario struct {
	Type         string         `json:"type"`
	SourceNodeID string         `json:"sourceNodeId,omitempty"`
	Target       string         `json:"target,omitempty"`
	Method       string         `json:"method,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// StartRequest — тело POST /api/v1/simulations.
type StartRequest struct {
	ProjectID  string   `json:"projectId"`
	TopologyID string   `json:"topologyId"`
	Scenario   Scenario `json:"scenario"`
	Seed       *int64   `json:"seed"`
}

// Simulation — запуск симуляции в ответах. Scenario — как сохранён (jsonb).
type Simulation struct {
	ID           string                 `json:"id"`
	ProjectID    string                 `json:"projectId"`
	TopologyID   string                 `json:"topologyId"`
	UserID       string                 `json:"userId"`
	Status       model.SimulationStatus `json:"status"`
	Scenario     json.RawMessage        `json:"scenario"`
	Seed         int64                  `json:"seed"`
	StartedAt    *time.Time             `json:"startedAt,omitempty"`
	FinishedAt   *time.Time             `json:"finishedAt,omitempty"`
	ErrorMessage *string                `json:"errorMessage,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

// StartResponse — ответ запуска: запуск, события и сводка — всё, что нужно
// фронтенду для Timeline без второго запроса.
type StartResponse struct {
	Simulation Simulation    `json:"simulation"`
	Events     []model.Event `json:"events"`
	Summary    model.Summary `json:"summary"`
}

// SimulationResponse — ответ GET /api/v1/simulations/{simulationId}.
type SimulationResponse struct {
	Simulation Simulation `json:"simulation"`
}

// EventsResponse — ответ GET /api/v1/simulations/{simulationId}/events.
type EventsResponse struct {
	Events []model.Event `json:"events"`
}

// StreamMessage — кадр WebSocket с одним событием симуляции.
// Поля в алфавитном порядке, как в прежнем кадре (map[string]any).
type StreamMessage struct {
	Event        model.Event `json:"event"`
	SimulationID string      `json:"simulationId"`
	Type         string      `json:"type"`
}
