package engine

import (
	"fmt"
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// Источники списка портов в config сервера (поле source в serverPort).
const (
	portSourceOpenPorts   = "openPorts"   // текущий формат
	portSourcePorts       = "ports"       // синоним openPorts
	portSourcePort        = "port"        // устаревший: один порт числом
	portSourceServicePort = "servicePort" // устаревший: один порт числом
	portStatusOpenValue   = "open"
)

// serverPort — открытый порт сервера в нормализованном виде. Уходит клиенту
// в details событий server.port.open/closed, поэтому с json-тегами.
type serverPort struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	Service  string `json:"service,omitempty"`
	Status   string `json:"status,omitempty"`
	Source   string `json:"source,omitempty"`
}

// serverPortCheckResult — слушает ли сервер нужный порт и почему.
type serverPortCheckResult struct {
	Open bool
	// ImplicitOpen — портов в config нет совсем, и сервер считается
	// слушающим любой порт: так ведут себя топологии до появления openPorts.
	ImplicitOpen  bool
	RequestedPort int
	Protocol      string
	OpenPorts     []serverPort
	MatchedPort   *serverPort // nil — подходящего порта нет (или порт открыт неявно)
}

// ensureServerPortOpen проверяет порт сервера и пишет результат в Timeline:
// server.port.open (и строку в Decisions) или server.port.closed с провалом симуляции.
func (r *runner) ensureServerPortOpen(node model.Node, protocol string, port int, sourceID, targetID string) bool {
	check := serverPortCheck(node, protocol, port)
	details := map[string]any{
		"nodeId":          node.ID,
		"nodeName":        nodeName(node),
		"protocol":        check.Protocol,
		"port":            check.RequestedPort,
		"open":            check.Open,
		"implicitOpen":    check.ImplicitOpen,
		"openPorts":       check.OpenPorts,
		"matchedOpenPort": check.MatchedPort,
	}

	if check.Open {
		r.emit(model.EventServerPortOpen, model.EventSeverityInfo, "server port is open", sourceID, targetID, r.packetID, details)
		r.summary.Decisions = append(r.summary.Decisions, fmt.Sprintf("%s listens on %s/%d", nodeName(node), check.Protocol, check.RequestedPort))

		return true
	}

	r.emit(model.EventServerPortClosed, model.EventSeverityError, "server port is closed", sourceID, targetID, r.packetID, details)
	r.fail(fmt.Sprintf("server does not listen on %s/%d", check.Protocol, check.RequestedPort))

	return false
}

// serverPortCheck ищет среди портов сервера открытый порт с нужными
// протоколом и номером.
func serverPortCheck(node model.Node, protocol string, port int) serverPortCheckResult {
	normalizedProtocol := strings.ToLower(strings.TrimSpace(protocol))
	if normalizedProtocol == "" {
		normalizedProtocol = httpsProtocol
	}

	ports, explicit := serverOpenPorts(node)
	result := serverPortCheckResult{RequestedPort: port, Protocol: normalizedProtocol, OpenPorts: ports}

	if len(ports) == 0 && !explicit {
		result.Open = true
		result.ImplicitOpen = true

		return result
	}

	for i := range ports {
		item := ports[i]
		if item.Protocol == normalizedProtocol && item.Port == port && portStatusOpen(item.Status) {
			result.Open = true
			result.MatchedPort = &item

			return result
		}
	}

	return result
}

// serverOpenPorts собирает порты сервера из всех поддерживаемых форматов.
//
// explicit=true значит «список портов задан явно», даже если он пустой или
// все элементы в нём кривые: тогда сервер не слушает ничего. Устаревшие
// port/servicePort читаются, только если в openPorts/ports не нашлось ни одного порта.
func serverOpenPorts(node model.Node) ([]serverPort, bool) {
	ports := make([]serverPort, 0)

	_, hasOpenPorts := node.Config[portSourceOpenPorts]
	for _, item := range anySlice(node.Config[portSourceOpenPorts]) {
		if parsed, ok := parseServerPort(item, portSourceOpenPorts); ok {
			ports = append(ports, parsed)
		}
	}

	_, hasPorts := node.Config[portSourcePorts]
	for _, item := range anySlice(node.Config[portSourcePorts]) {
		if parsed, ok := parseServerPort(item, portSourcePorts); ok {
			ports = append(ports, parsed)
		}
	}

	if len(ports) > 0 {
		return ports, true
	}

	if port := intValue(node.Config[portSourcePort], 0); port > 0 {
		ports = append(ports, legacyServerPort(node, port, portSourcePort))
	}

	if port := intValue(node.Config[portSourceServicePort], 0); port > 0 {
		ports = append(ports, legacyServerPort(node, port, portSourceServicePort))
	}

	return ports, hasOpenPorts || hasPorts || len(ports) > 0
}

// parseServerPort разбирает элемент списка портов: число (443 → tcp/443)
// или объект {"port", "protocol", "service"/"name", "status"}.
// Протокол по умолчанию tcp, статус — open, имя сервиса — из известных портов.
func parseServerPort(value any, source string) (serverPort, bool) {
	if port := intValue(value, 0); port > 0 {
		return serverPort{Protocol: httpsProtocol, Port: port, Service: wellKnownService(port), Status: portStatusOpenValue, Source: source}, true
	}

	m := anyMap(value)

	port := intValue(m["port"], 0)
	if port <= 0 {
		return serverPort{}, false
	}

	protocol := strings.ToLower(strings.TrimSpace(stringValue(m["protocol"])))
	if protocol == "" {
		protocol = httpsProtocol
	}

	service := strings.TrimSpace(stringValue(m["service"]))
	if service == "" {
		service = strings.TrimSpace(stringValue(m["name"]))
	}

	if service == "" {
		service = wellKnownService(port)
	}

	status := strings.ToLower(strings.TrimSpace(stringValue(m["status"])))
	if status == "" {
		status = portStatusOpenValue
	}

	return serverPort{Protocol: protocol, Port: port, Service: service, Status: status, Source: source}, true
}

// legacyServerPort — порт из устаревших полей port/servicePort: всегда tcp
// и всегда открыт; имя сервиса — из serviceName или известных портов.
func legacyServerPort(node model.Node, port int, source string) serverPort {
	service := strings.TrimSpace(stringValue(node.Config["serviceName"]))
	if service == "" {
		service = wellKnownService(port)
	}

	return serverPort{Protocol: httpsProtocol, Port: port, Service: service, Status: portStatusOpenValue, Source: source}
}

// portStatusOpen считает открытыми порты со статусом open, active, healthy
// или без статуса.
func portStatusOpen(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", portStatusOpenValue, "active", "healthy":
		return true
	default:
		return false
	}
}

// wellKnownService — имя сервиса по номеру порта для подписей в инспекторе.
func wellKnownService(port int) string {
	switch port {
	case 22:
		return "SSH"
	case 53:
		return "DNS"
	case 80:
		return "HTTP"
	case 443:
		return "HTTPS"
	case 5432:
		return "PostgreSQL"
	case 6379:
		return "Redis"
	default:
		return ""
	}
}
