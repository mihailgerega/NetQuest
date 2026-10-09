package model

// Summary — сводка симуляции для Packet Inspector и проверок квестов.
//
// Копия сводки вкладывается в details событий simulation.completed и
// simulation.failed и вместе с ними сохраняется в jsonb, поэтому json-теги —
// формат хранения: старые события в базе читаются фронтендом по тем же ключам.
type Summary struct {
	PacketID              string           `json:"packetId,omitempty"`
	Scenario              string           `json:"scenario"`
	Status                SimulationStatus `json:"status"`
	Seed                  int64            `json:"seed"`
	Source                string           `json:"source,omitempty"`
	SourceNodeID          string           `json:"sourceNodeId,omitempty"`
	Destination           string           `json:"destination,omitempty"`
	ResolvedIP            string           `json:"resolvedIp,omitempty"`
	SelectedBackend       string           `json:"selectedBackend,omitempty"`
	SelectedBackendNodeID string           `json:"selectedBackendNodeId,omitempty"`
	SelectedBackendName   string           `json:"selectedBackendName,omitempty"`
	HealthyBackends       []string         `json:"healthyBackends,omitempty"`
	SkippedBackends       []BackendSkip    `json:"skippedBackends,omitempty"`
	Failover              bool             `json:"failover"`
	TotalLatencyMs        int64            `json:"totalLatencyMs"`
	LatencyBreakdown      []LatencyStage   `json:"latencyBreakdown,omitempty"`
	LatencyFormula        string           `json:"latencyFormula,omitempty"`
	ProtocolDetails       ProtocolDetails  `json:"protocolDetails,omitempty"`
	Path                  []string         `json:"path"`
	Decisions             []string         `json:"decisions"`
	Errors                []string         `json:"errors"`
	Metadata              map[string]any   `json:"metadata,omitempty"`
}

// LatencyStage — один этап в разборе задержки: из суммы этапов
// фронтенд объясняет, откуда взялся totalLatencyMs.
type LatencyStage struct {
	Stage      string         `json:"stage"`
	Label      string         `json:"label"`
	DurationMs int64          `json:"durationMs"`
	Details    map[string]any `json:"details,omitempty"`
}

// ProtocolDetails — протокольный разбор по слоям (DNS, маршрутизация, firewall,
// TCP, TLS, сервер, Load Balancer). Собирается из событий в конце симуляции.
type ProtocolDetails struct {
	Summary      map[string]any   `json:"summary,omitempty"`
	DNS          map[string]any   `json:"dns,omitempty"`
	Routing      map[string]any   `json:"routing,omitempty"`
	Firewall     map[string]any   `json:"firewall,omitempty"`
	TCP          map[string]any   `json:"tcp,omitempty"`
	TLS          map[string]any   `json:"tls,omitempty"`
	Server       map[string]any   `json:"server,omitempty"`
	LoadBalancer map[string]any   `json:"loadBalancer,omitempty"`
	Errors       []map[string]any `json:"errors,omitempty"`
}

// BackendSkip — сервер, который Load Balancer исключил из выбора, и причина.
type BackendSkip struct {
	NodeID string `json:"nodeId"`
	Name   string `json:"name,omitempty"`
	Reason string `json:"reason"`
}
