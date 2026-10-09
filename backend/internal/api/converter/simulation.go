package converter

import (
	simulationv1 "github.com/netquest/netquest/backend/internal/contract/simulation/v1"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// ScenarioToModel переводит сценарий из запроса в доменный.
func ScenarioToModel(scenario simulationv1.Scenario) model.Scenario {
	return model.Scenario{
		Type:         scenario.Type,
		SourceNodeID: scenario.SourceNodeID,
		Target:       scenario.Target,
		Method:       scenario.Method,
		Metadata:     scenario.Metadata,
	}
}

// OptionalScenarioToModel — то же для необязательного сценария; nil остаётся nil.
func OptionalScenarioToModel(scenario *simulationv1.Scenario) *model.Scenario {
	if scenario == nil {
		return nil
	}

	converted := ScenarioToModel(*scenario)

	return &converted
}

// ToStartSimulationInput переводит запрос запуска во вход сервиса.
func ToStartSimulationInput(req simulationv1.StartRequest) input.StartSimulationInput {
	return input.StartSimulationInput{
		ProjectID:  req.ProjectID,
		TopologyID: req.TopologyID,
		Scenario:   ScenarioToModel(req.Scenario),
		Seed:       req.Seed,
	}
}

// SimulationToDTO переводит запуск симуляции в DTO ответа.
func SimulationToDTO(simulation model.Simulation) simulationv1.Simulation {
	return simulationv1.Simulation{
		ID:           simulation.ID,
		ProjectID:    simulation.ProjectID,
		TopologyID:   simulation.TopologyID,
		UserID:       simulation.UserID,
		Status:       simulation.Status,
		Scenario:     simulation.Scenario,
		Seed:         simulation.Seed,
		StartedAt:    simulation.StartedAt,
		FinishedAt:   simulation.FinishedAt,
		ErrorMessage: simulation.ErrorMessage,
		CreatedAt:    simulation.CreatedAt,
		UpdatedAt:    simulation.UpdatedAt,
	}
}
