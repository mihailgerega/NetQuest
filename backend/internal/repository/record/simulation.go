package record

import "time"

// Simulation — строка таблицы simulations. scenario (jsonb) читается текстом.
type Simulation struct {
	ID           string     `db:"id"`
	ProjectID    string     `db:"project_id"`
	TopologyID   string     `db:"topology_id"`
	UserID       string     `db:"user_id"`
	Status       string     `db:"status"`
	Scenario     string     `db:"scenario"`
	Seed         int64      `db:"seed"`
	StartedAt    *time.Time `db:"started_at"`    // NULL → nil
	FinishedAt   *time.Time `db:"finished_at"`   // NULL → nil
	ErrorMessage *string    `db:"error_message"` // NULL → nil
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

// SimulationEvent — строка таблицы simulation_events. Необязательные узлы
// и пакет читаются через COALESCE в пустую строку, details (jsonb) — текстом.
type SimulationEvent struct {
	ID             string `db:"id"`
	SimulationID   string `db:"simulation_id"`
	SequenceNumber int64  `db:"sequence_number"`
	TimestampMs    int64  `db:"timestamp_ms"`
	Type           string `db:"type"`
	Severity       string `db:"severity"`
	PacketID       string `db:"packet_id"`
	SourceNodeID   string `db:"source_node_id"`
	TargetNodeID   string `db:"target_node_id"`
	Message        string `db:"message"`
	Details        string `db:"details"`
}
