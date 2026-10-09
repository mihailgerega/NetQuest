package input

import "github.com/netquest/netquest/backend/internal/model"

// StartSimulationInput — вход запуска симуляции.
type StartSimulationInput struct {
	ProjectID  string
	TopologyID string
	Scenario   model.Scenario
	Seed       *int64 // nil — seed из текущего времени
}
