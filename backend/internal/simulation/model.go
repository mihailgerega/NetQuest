package simulation

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type EventType string

const (
	EventSimulationStarted    EventType = "simulation.started"
	EventTopologyValidated    EventType = "topology.validated"
	EventPacketCreated        EventType = "packet.created"
	EventDNSQuery             EventType = "dns.query"
	EventDNSResponse          EventType = "dns.response"
	EventDNSError             EventType = "dns.error"
	EventRouteSelected        EventType = "route.selected"
	EventRouteNotFound        EventType = "route.not_found"
	EventFirewallDecision     EventType = "firewall.decision"
	EventFirewallDenied       EventType = "firewall.denied"
	EventTCPHandshakeStart    EventType = "tcp.handshake.start"
	EventTCPSYN               EventType = "tcp.syn"
	EventTCPSYNACK            EventType = "tcp.syn_ack"
	EventTCPACK               EventType = "tcp.ack"
	EventTCPHandshakeDone     EventType = "tcp.handshake.done"
	EventTLSHandshakeStart    EventType = "tls.handshake.start"
	EventTLSClientHello       EventType = "tls.client_hello"
	EventTLSServerHello       EventType = "tls.server_hello"
	EventTLSCertValidated     EventType = "tls.certificate.validated"
	EventTLSHandshakeDone     EventType = "tls.handshake.done"
	EventLBBackendDiscovered  EventType = "lb.backend.discovered"
	EventLBBackendSelected    EventType = "lb.backend.selected"
	EventLBBackendUnhealthy   EventType = "lb.backend.unhealthy"
	EventPacketForwarded      EventType = "packet.forwarded"
	EventPacketDropped        EventType = "packet.dropped"
	EventPacketDelivered      EventType = "packet.delivered"
	EventFailoverTriggered    EventType = "failover.triggered"
	EventFailoverRouteChanged EventType = "failover.route_changed"
	EventSimulationCompleted  EventType = "simulation.completed"
	EventSimulationFailed     EventType = "simulation.failed"
)

type Protocol string

const (
	ProtocolVirtual Protocol = "virtual"
)

type PacketState string

const (
	PacketStateCreated   PacketState = "created"
	PacketStateDelivered PacketState = "delivered"
)

type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

type Scenario struct {
	Type         string         `json:"type"`
	SourceNodeID string         `json:"sourceNodeId,omitempty"`
	Target       string         `json:"target,omitempty"`
	Method       string         `json:"method,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type Destination struct {
	NodeID string `json:"nodeId,omitempty"`
	Host   string `json:"host,omitempty"`
}

type Hop struct {
	NodeID      string `json:"nodeId"`
	TimestampMs int64  `json:"timestampMs"`
}

type Packet struct {
	ID              string         `json:"id"`
	Protocol        Protocol       `json:"protocol"`
	SourceNodeID    string         `json:"sourceNodeId"`
	Destination     Destination    `json:"destination"`
	SourceIP        string         `json:"sourceIp,omitempty"`
	DestinationIP   string         `json:"destinationIp,omitempty"`
	SourcePort      int            `json:"sourcePort,omitempty"`
	DestinationPort int            `json:"destinationPort,omitempty"`
	TTL             int            `json:"ttl"`
	SizeBytes       int            `json:"sizeBytes"`
	State           PacketState    `json:"state"`
	Path            []Hop          `json:"path"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type Event struct {
	ID             string         `json:"id"`
	SimulationID   string         `json:"simulationId"`
	SequenceNumber int64          `json:"sequenceNumber"`
	Type           EventType      `json:"type"`
	TimestampMs    int64          `json:"timestampMs"`
	SourceNodeID   string         `json:"sourceNodeId,omitempty"`
	TargetNodeID   string         `json:"targetNodeId,omitempty"`
	PacketID       string         `json:"packetId,omitempty"`
	Severity       Severity       `json:"severity"`
	Message        string         `json:"message"`
	Details        map[string]any `json:"details"`
}

type RunRequest struct {
	SimulationID string
	Topology     json.RawMessage
	Scenario     Scenario
	Seed         int64
}

type RunResult struct {
	Status  Status  `json:"status"`
	Seed    int64   `json:"seed"`
	Events  []Event `json:"events"`
	Summary Summary `json:"summary"`
}

type Summary struct {
	PacketID              string          `json:"packetId,omitempty"`
	Scenario              string          `json:"scenario"`
	Status                Status          `json:"status"`
	Seed                  int64           `json:"seed"`
	Source                string          `json:"source,omitempty"`
	SourceNodeID          string          `json:"sourceNodeId,omitempty"`
	Destination           string          `json:"destination,omitempty"`
	ResolvedIP            string          `json:"resolvedIp,omitempty"`
	SelectedBackend       string          `json:"selectedBackend,omitempty"`
	SelectedBackendNodeID string          `json:"selectedBackendNodeId,omitempty"`
	SelectedBackendName   string          `json:"selectedBackendName,omitempty"`
	HealthyBackends       []string        `json:"healthyBackends,omitempty"`
	SkippedBackends       []BackendSkip   `json:"skippedBackends,omitempty"`
	Failover              bool            `json:"failover"`
	TotalLatencyMs        int64           `json:"totalLatencyMs"`
	LatencyBreakdown      []LatencyStage  `json:"latencyBreakdown,omitempty"`
	LatencyFormula        string          `json:"latencyFormula,omitempty"`
	ProtocolDetails       ProtocolDetails `json:"protocolDetails,omitempty"`
	Path                  []string        `json:"path"`
	Decisions             []string        `json:"decisions"`
	Errors                []string        `json:"errors"`
	Metadata              map[string]any  `json:"metadata,omitempty"`
}

type LatencyStage struct {
	Stage      string         `json:"stage"`
	Label      string         `json:"label"`
	DurationMs int64          `json:"durationMs"`
	Details    map[string]any `json:"details,omitempty"`
}

type ProtocolDetails struct {
	Summary      map[string]any   `json:"summary,omitempty"`
	DNS          map[string]any   `json:"dns,omitempty"`
	Routing      map[string]any   `json:"routing,omitempty"`
	Firewall     map[string]any   `json:"firewall,omitempty"`
	TCP          map[string]any   `json:"tcp,omitempty"`
	TLS          map[string]any   `json:"tls,omitempty"`
	LoadBalancer map[string]any   `json:"loadBalancer,omitempty"`
	Errors       []map[string]any `json:"errors,omitempty"`
}

type BackendSkip struct {
	NodeID string `json:"nodeId"`
	Name   string `json:"name,omitempty"`
	Reason string `json:"reason"`
}

type Simulation struct {
	ID           string          `json:"id"`
	ProjectID    string          `json:"projectId"`
	TopologyID   string          `json:"topologyId"`
	UserID       string          `json:"userId"`
	Status       Status          `json:"status"`
	Scenario     json.RawMessage `json:"scenario"`
	Seed         int64           `json:"seed"`
	StartedAt    *time.Time      `json:"startedAt,omitempty"`
	FinishedAt   *time.Time      `json:"finishedAt,omitempty"`
	ErrorMessage *string         `json:"errorMessage,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type StartRequest struct {
	ProjectID  string   `json:"projectId"`
	TopologyID string   `json:"topologyId"`
	Scenario   Scenario `json:"scenario"`
	Seed       *int64   `json:"seed"`
}
