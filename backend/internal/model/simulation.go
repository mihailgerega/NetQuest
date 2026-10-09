package model

import (
	"encoding/json"
	"time"
)

// SimulationStatus — состояние запуска симуляции. Хранится в simulations.status.
type SimulationStatus string

// Жизненный цикл: pending (строка создана) → running (движок считает) →
// completed или failed (итог движка).
const (
	SimulationStatusPending   SimulationStatus = "pending"
	SimulationStatusRunning   SimulationStatus = "running"
	SimulationStatusCompleted SimulationStatus = "completed"
	SimulationStatusFailed    SimulationStatus = "failed"
)

// Типы сценариев, которые умеет считать движок (Scenario.Type).
const (
	ScenarioDNSLookup    = "dns_lookup"
	ScenarioICMPPing     = "icmp_ping"
	ScenarioHTTPSRequest = "https_request"
	ScenarioFailoverDemo = "failover_demo"
)

// Scenario — что именно симулировать: откуда (SourceNodeID — всегда client),
// куда (Target — hostname, URL, IP или ID узла) и каким протоколом (Type).
//
// Сохраняется в simulations.scenario как jsonb, поэтому json-теги — часть
// формата хранения. Имя типа тоже значимо: encoding/json вставляет его в текст
// ошибки разбора ("Go struct field Scenario.scenario.type"), а этот текст
// уходит клиенту в ответе 400.
type Scenario struct {
	Type         string         `json:"type"`
	SourceNodeID string         `json:"sourceNodeId,omitempty"`
	Target       string         `json:"target,omitempty"`
	Method       string         `json:"method,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// Simulation — один запуск сценария над конкретной версией топологии.
//
// Seed делает запуск воспроизводимым: с тем же seed, топологией и сценарием
// движок выдаст те же события, задержки и выбор сервера.
type Simulation struct {
	ID           string
	ProjectID    string
	TopologyID   string
	UserID       string
	Status       SimulationStatus
	Scenario     json.RawMessage
	Seed         int64
	StartedAt    *time.Time // nil — движок ещё не запускался
	FinishedAt   *time.Time // nil — движок ещё не закончил
	ErrorMessage *string    // nil — без ошибки
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// RunRequest — вход движка симуляции: что и над какой топологией считать.
type RunRequest struct {
	SimulationID string
	Topology     json.RawMessage
	Scenario     Scenario
	Seed         int64
}

// RunResult — выход движка: итоговый статус, события по порядку и сводка.
type RunResult struct {
	Status  SimulationStatus `json:"status"`
	Seed    int64            `json:"seed"`
	Events  []Event          `json:"events"`
	Summary Summary          `json:"summary"`
}
