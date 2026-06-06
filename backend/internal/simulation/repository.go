package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/netquest/netquest/backend/pkg/apperrors"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, simulation Simulation) (Simulation, error) {
	if r.db == nil {
		return Simulation{}, apperrors.Internal("postgres is not configured")
	}

	var scenarioText string
	err := r.db.QueryRow(ctx, `
		INSERT INTO simulations (id, project_id, topology_id, user_id, status, scenario, seed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, now(), now())
		RETURNING id::text, project_id::text, topology_id::text, user_id::text, status, scenario::text,
		          seed, started_at, finished_at, error_message, created_at, updated_at
	`, simulation.ID, simulation.ProjectID, simulation.TopologyID, simulation.UserID, simulation.Status, string(simulation.Scenario), simulation.Seed).Scan(
		&simulation.ID,
		&simulation.ProjectID,
		&simulation.TopologyID,
		&simulation.UserID,
		&simulation.Status,
		&scenarioText,
		&simulation.Seed,
		&simulation.StartedAt,
		&simulation.FinishedAt,
		&simulation.ErrorMessage,
		&simulation.CreatedAt,
		&simulation.UpdatedAt,
	)
	if err != nil {
		return Simulation{}, fmt.Errorf("create simulation: %w", err)
	}
	simulation.Scenario = json.RawMessage(scenarioText)
	return simulation, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status Status, errorMessage *string, startedAt, finishedAt *time.Time) error {
	if r.db == nil {
		return apperrors.Internal("postgres is not configured")
	}

	_, err := r.db.Exec(ctx, `
		UPDATE simulations
		SET status = $2,
		    error_message = $3,
		    started_at = COALESCE($4, started_at),
		    finished_at = COALESCE($5, finished_at),
		    updated_at = now()
		WHERE id = $1
	`, id, status, errorMessage, startedAt, finishedAt)
	if err != nil {
		return fmt.Errorf("update simulation status: %w", err)
	}
	return nil
}

func (r *Repository) InsertEvents(ctx context.Context, simulationID string, events []Event) error {
	if r.db == nil {
		return apperrors.Internal("postgres is not configured")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin insert simulation events: %w", err)
	}
	defer tx.Rollback(ctx)

	for i, event := range events {
		sequence := event.SequenceNumber
		if sequence == 0 {
			sequence = int64(i + 1)
		}
		details, err := json.Marshal(event.Details)
		if err != nil {
			return fmt.Errorf("marshal event details: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO simulation_events (
				id, simulation_id, sequence_number, timestamp_ms, type, severity, packet_id,
				source_node_id, target_node_id, message, details, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), $10, $11::jsonb, now())
		`, event.ID, simulationID, sequence, event.TimestampMs, event.Type, event.Severity, event.PacketID, event.SourceNodeID, event.TargetNodeID, event.Message, string(details))
		if err != nil {
			return fmt.Errorf("insert simulation event: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit simulation events: %w", err)
	}
	return nil
}

func (r *Repository) GetForOwner(ctx context.Context, simulationID, ownerID string) (Simulation, error) {
	if r.db == nil {
		return Simulation{}, apperrors.Internal("postgres is not configured")
	}

	var simulation Simulation
	var scenarioText string
	err := r.db.QueryRow(ctx, `
		SELECT s.id::text, s.project_id::text, s.topology_id::text, s.user_id::text, s.status,
		       s.scenario::text, s.seed, s.started_at, s.finished_at, s.error_message, s.created_at, s.updated_at
		FROM simulations s
		INNER JOIN projects p ON p.id = s.project_id
		WHERE s.id = $1 AND p.owner_id = $2 AND s.deleted_at IS NULL AND p.deleted_at IS NULL
	`, simulationID, ownerID).Scan(
		&simulation.ID,
		&simulation.ProjectID,
		&simulation.TopologyID,
		&simulation.UserID,
		&simulation.Status,
		&scenarioText,
		&simulation.Seed,
		&simulation.StartedAt,
		&simulation.FinishedAt,
		&simulation.ErrorMessage,
		&simulation.CreatedAt,
		&simulation.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Simulation{}, apperrors.NotFound("simulation not found")
	}
	if err != nil {
		return Simulation{}, fmt.Errorf("get simulation: %w", err)
	}
	simulation.Scenario = json.RawMessage(scenarioText)
	return simulation, nil
}

func (r *Repository) ListEventsForOwner(ctx context.Context, simulationID, ownerID string) ([]Event, error) {
	if r.db == nil {
		return nil, apperrors.Internal("postgres is not configured")
	}

	rows, err := r.db.Query(ctx, `
		SELECT e.id::text, e.simulation_id::text, e.sequence_number, e.timestamp_ms, e.type, e.severity,
		       COALESCE(e.packet_id, ''), COALESCE(e.source_node_id, ''), COALESCE(e.target_node_id, ''),
		       e.message, e.details::text
		FROM simulation_events e
		INNER JOIN simulations s ON s.id = e.simulation_id
		INNER JOIN projects p ON p.id = s.project_id
		WHERE e.simulation_id = $1 AND p.owner_id = $2 AND s.deleted_at IS NULL AND p.deleted_at IS NULL
		ORDER BY e.sequence_number ASC
	`, simulationID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list simulation events: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		var event Event
		var detailsText string
		if err := rows.Scan(
			&event.ID,
			&event.SimulationID,
			&event.SequenceNumber,
			&event.TimestampMs,
			&event.Type,
			&event.Severity,
			&event.PacketID,
			&event.SourceNodeID,
			&event.TargetNodeID,
			&event.Message,
			&detailsText,
		); err != nil {
			return nil, fmt.Errorf("scan simulation event: %w", err)
		}
		if err := json.Unmarshal([]byte(detailsText), &event.Details); err != nil {
			return nil, fmt.Errorf("decode simulation event details: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate simulation events: %w", err)
	}
	return events, nil
}
