package simulation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/netquest/netquest/backend/internal/model"
)

// InsertEvents сохраняет события симуляции одной транзакцией: либо все,
// либо ни одного — Timeline не должен остаться с дырой посередине.
//
// Номер события без SequenceNumber — его позиция в срезе (с единицы).
// Пустые packet_id и ID узлов пишутся как NULL.
func (r *repository) InsertEvents(ctx context.Context, simulationID string, events []model.Event) error {
	const query = `
		INSERT INTO simulation_events (
			id, simulation_id, sequence_number, timestamp_ms, type, severity, packet_id,
			source_node_id, target_node_id, message, details, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), $10, $11::jsonb, now())`

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		for i, event := range events {
			sequence := event.SequenceNumber
			if sequence == 0 {
				sequence = int64(i + 1)
			}

			details, err := json.Marshal(event.Details)
			if err != nil {
				return fmt.Errorf("сериализовать details события: %w", err)
			}

			if _, err := tx.Exec(ctx, query,
				event.ID, simulationID, sequence, event.TimestampMs, event.Type, event.Severity, event.PacketID,
				event.SourceNodeID, event.TargetNodeID, event.Message, string(details)); err != nil {
				return fmt.Errorf("вставить событие симуляции: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("сохранить события симуляции: %w", err)
	}

	return nil
}
