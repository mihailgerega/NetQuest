package model

// EventType — тип события симуляции. Фронтенд строит по нему Timeline
// и подсвечивает путь пакета, поэтому значения — часть контракта.
type EventType string

// События симуляции в порядке, в котором они обычно появляются в Timeline.
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
	EventServerPortOpen       EventType = "server.port.open"
	EventServerPortClosed     EventType = "server.port.closed"
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

// EventSeverity — важность события в Timeline.
type EventSeverity string

// Значения важности: warn — сеть справилась, но не по основному пути
// (повтор, пропущенный сервер); error — шаг сценария провалился.
const (
	EventSeverityInfo  EventSeverity = "info"
	EventSeverityWarn  EventSeverity = "warn"
	EventSeverityError EventSeverity = "error"
)

// Event — одно событие симуляции.
//
// TimestampMs — виртуальное время от начала симуляции, а не время сервера:
// движок сам двигает часы на задержки каналов и обработки. SequenceNumber задаёт
// порядок событий с одинаковым временем. Details — подробности шага, у каждого
// типа события свой набор ключей; в базе лежат в simulation_events.details (jsonb).
type Event struct {
	ID             string         `json:"id"`
	SimulationID   string         `json:"simulationId"`
	SequenceNumber int64          `json:"sequenceNumber"`
	Type           EventType      `json:"type"`
	TimestampMs    int64          `json:"timestampMs"`
	SourceNodeID   string         `json:"sourceNodeId,omitempty"`
	TargetNodeID   string         `json:"targetNodeId,omitempty"`
	PacketID       string         `json:"packetId,omitempty"`
	Severity       EventSeverity  `json:"severity"`
	Message        string         `json:"message"`
	Details        map[string]any `json:"details"`
}
