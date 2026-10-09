package converter

import (
	"encoding/json"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// SimulationToModel переводит строку simulations в доменную модель.
func SimulationToModel(rec record.Simulation) model.Simulation {
	return model.Simulation{
		ID:           rec.ID,
		ProjectID:    rec.ProjectID,
		TopologyID:   rec.TopologyID,
		UserID:       rec.UserID,
		Status:       model.SimulationStatus(rec.Status),
		Scenario:     json.RawMessage(rec.Scenario),
		Seed:         rec.Seed,
		StartedAt:    rec.StartedAt,
		FinishedAt:   rec.FinishedAt,
		ErrorMessage: rec.ErrorMessage,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
	}
}

// SimulationEventsToModel переводит строки simulation_events в события.
//
// details хранится jsonb и разбирается обратно в map: ошибка разбора —
// испорченная строка в базе, это сбой, а не пустые подробности.
func SimulationEventsToModel(recs []record.SimulationEvent) ([]model.Event, error) {
	events := make([]model.Event, 0, len(recs))

	for _, rec := range recs {
		event := model.Event{
			ID:             rec.ID,
			SimulationID:   rec.SimulationID,
			SequenceNumber: rec.SequenceNumber,
			Type:           model.EventType(rec.Type),
			TimestampMs:    rec.TimestampMs,
			SourceNodeID:   rec.SourceNodeID,
			TargetNodeID:   rec.TargetNodeID,
			PacketID:       rec.PacketID,
			Severity:       model.EventSeverity(rec.Severity),
			Message:        rec.Message,
		}

		if err := json.Unmarshal([]byte(rec.Details), &event.Details); err != nil {
			return nil, fmt.Errorf("разобрать details события %s: %w", rec.ID, err)
		}

		events = append(events, event)
	}

	return events, nil
}
