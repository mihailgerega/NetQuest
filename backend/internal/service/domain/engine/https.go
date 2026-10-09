package engine

import (
	"net"

	"github.com/netquest/netquest/backend/internal/model"
)

// HTTPS всегда идёт на tcp/443: порт и протокол проверяют firewall и Server.
const (
	httpsProtocol = "tcp"
	httpsPort     = 443
)

// runHTTPS — сценарии https_request и failover_demo.
//
// Путь запроса:
//
//	DNS (если цель — имя, а не IP) → узел с полученным IP → маршрут
//	→ [failover] → потеря пакета → firewall → открыт ли tcp/443 у Server
//	→ TCP handshake → TLS handshake → [выбор сервера Load Balancer'ом] → доставка
//
// Каждый шаг может провалить симуляцию; тогда следующие шаги не выполняются.
func (r *runner) runHTTPS(failover bool) {
	source := r.req.Scenario.SourceNodeID
	hostname := targetHost(r.req.Scenario.Target)

	resolvedIP, ok := r.resolveHTTPSTarget(hostname)
	if !ok {
		return
	}

	destination, ok := r.findNodeByIP(resolvedIP)
	if !ok {
		r.fail("resolved IP does not belong to a topology node: " + resolvedIP)
		return
	}

	path, latency, ok := r.routeOrFail(source, destination.ID)
	if !ok {
		return
	}

	r.failoverIfNeeded(failover, source, destination.ID, path)
	r.selectRoute(source, destination.ID, path, latency)
	r.summary.Decisions = append(r.summary.Decisions, "Graph route selected to "+destination.ID)

	if !r.survivePacketLoss(source, destination.ID, path, latency, packetLossMessages{
		firstDrop:  "packet dropped before TCP handshake",
		secondDrop: "packet retry dropped before TCP handshake",
	}) {
		return
	}

	if !r.firewallAllows(path, resolvedIP, httpsPort) {
		return
	}

	if destination.Type == model.NodeTypeServer && !r.ensureServerPortOpen(destination, httpsProtocol, httpsPort, source, destination.ID) {
		return
	}

	r.tcpHandshake(source, destination.ID, path, latency)
	r.tlsHandshake(source, destination.ID, hostname, path, latency)

	finalPath, selectedBackend, ok := r.deliverHTTPS(destination, path, latency)
	if !ok {
		return
	}

	r.summary.Path = finalPath
	r.summary.LatencyFormula = "HTTPS latency ≈ DNS + route lookup + firewall decision + TCP handshake + TLS handshake + Load Balancer decision + delivery"
	r.summary.TotalLatencyMs = r.timestamp

	if selectedBackend != "" {
		r.summary.Decisions = append(r.summary.Decisions, "Load balancer selected "+r.summary.SelectedBackendName)
	}

	r.emit(model.EventPacketDelivered, model.EventSeverityInfo, "HTTPS request delivered", source, destination.ID, r.packetID, map[string]any{
		"path":            finalPath,
		"selectedBackend": selectedBackend,
		"latencyMs":       r.summary.TotalLatencyMs,
	})
	r.complete("HTTPS request completed")
}

// resolveHTTPSTarget превращает цель запроса в IP: имя разрешается через DNS,
// IP используется как есть.
func (r *runner) resolveHTTPSTarget(hostname string) (string, bool) {
	if net.ParseIP(hostname) != nil {
		r.summary.Decisions = append(r.summary.Decisions, "Target is already an IP address: "+hostname)
		r.summary.ResolvedIP = hostname

		return hostname, true
	}

	answer, ok := r.resolveDNS(hostname)
	if !ok {
		// resolveDNS уже провалил симуляцию с точной причиной; этот fail
		// ничего не перезапишет — первая ошибка окончательна.
		r.fail("DNS NXDOMAIN for " + hostname)
		return "", false
	}

	r.summary.ResolvedIP = answer.IP

	return answer.IP, true
}

// tcpHandshake — трёхстороннее рукопожатие TCP: SYN → SYN-ACK → ACK.
// Стоит два прохода по пути (RTT) плюс обработка.
func (r *runner) tcpHandshake(source, destinationID string, path []string, latency int64) {
	tcpStart := r.timestamp

	r.emit(model.EventTCPHandshakeStart, model.EventSeverityInfo, "TCP handshake started", source, destinationID, r.packetID, nil)
	r.emit(model.EventTCPSYN, model.EventSeverityInfo, "TCP SYN sent", source, destinationID, r.packetID, nil)
	r.advance(latency)
	r.emit(model.EventTCPSYNACK, model.EventSeverityInfo, "TCP SYN-ACK received", destinationID, source, r.packetID, nil)
	r.advance(latency)
	r.emit(model.EventTCPACK, model.EventSeverityInfo, "TCP ACK sent", source, destinationID, r.packetID, nil)

	tcpProcessing := r.processingDelay(1, 2)
	r.advance(tcpProcessing)
	r.emit(model.EventTCPHandshakeDone, model.EventSeverityInfo, "TCP handshake completed", source, destinationID, r.packetID, nil)

	r.addLatencyStage("tcp_handshake", "TCP Handshake", r.timestamp-tcpStart, map[string]any{
		"path":            path,
		"oneWayLatencyMs": latency,
		"rttMs":           latency * 2,
		"processingMs":    tcpProcessing,
	})
}

// tlsHandshake — виртуальное рукопожатие TLS 1.3 (один round trip).
// Настоящей криптографии нет: сертификат «проверяется» по имени хоста.
func (r *runner) tlsHandshake(source, destinationID, hostname string, path []string, latency int64) {
	tlsStart := r.timestamp

	tlsSetupProcessing := r.processingDelay(1, 3)
	r.advance(tlsSetupProcessing)
	r.emit(model.EventTLSHandshakeStart, model.EventSeverityInfo, "TLS handshake started", source, destinationID, r.packetID, nil)
	r.emit(model.EventTLSClientHello, model.EventSeverityInfo, "TLS ClientHello sent", source, destinationID, r.packetID, map[string]any{"serverName": hostname})
	r.advance(latency)
	r.emit(model.EventTLSServerHello, model.EventSeverityInfo, "TLS ServerHello received", destinationID, source, r.packetID, nil)

	tlsCertificateProcessing := r.processingDelay(2, 5)
	r.advance(tlsCertificateProcessing)
	r.emit(model.EventTLSCertValidated, model.EventSeverityInfo, "TLS certificate validated", destinationID, source, r.packetID, map[string]any{"hostname": hostname})
	r.advance(latency)
	r.emit(model.EventTLSHandshakeDone, model.EventSeverityInfo, "TLS handshake completed", source, destinationID, r.packetID, nil)

	r.addLatencyStage("tls_handshake", "TLS Handshake", r.timestamp-tlsStart, map[string]any{
		"path":            path,
		"oneWayLatencyMs": latency,
		"roundTrips":      1,
		"rttMs":           latency * 2,
		"processingMs":    tlsSetupProcessing + tlsCertificateProcessing,
	})
}

// deliverHTTPS доставляет запрос получателю. Если получатель — Load Balancer,
// он выбирает сервер, и путь продлевается от балансировщика до сервера.
// Возвращает итоговый путь и ID выбранного сервера (пусто без балансировщика).
func (r *runner) deliverHTTPS(destination model.Node, path []string, latency int64) ([]string, string, bool) {
	if destination.Type != model.NodeTypeLoadBalancer {
		deliveryStart := r.timestamp
		deliveryProcessing := r.processingDelay(1, 3)
		r.advance(latency + deliveryProcessing)
		r.addLatencyStage("packet_delivery", "Packet delivery", r.timestamp-deliveryStart, map[string]any{
			"path":            path,
			"oneWayLatencyMs": latency,
			"processingMs":    deliveryProcessing,
		})

		return path, "", true
	}

	lbStart := r.timestamp
	lbProcessing := r.processingDelay(1, 5)
	r.advance(lbProcessing)

	backend, ok := r.selectBackend(destination, httpsPort)
	if !ok {
		return nil, "", false
	}

	r.summary.SelectedBackend = backend.ID
	r.summary.SelectedBackendNodeID = backend.ID
	r.summary.SelectedBackendName = nodeName(backend)
	r.addLatencyStage("load_balancer_decision", "Load Balancer decision", r.timestamp-lbStart, map[string]any{
		"processingMs":          lbProcessing,
		"selectedBackendNodeId": backend.ID,
		"selectedBackendName":   nodeName(backend),
		"healthyBackends":       r.summary.HealthyBackends,
		"skippedBackends":       r.summary.SkippedBackends,
	})

	backendPath, backendLatency, ok := r.findPath(destination.ID, backend.ID)
	if !ok {
		r.fail("Load balancer has no healthy backends available.")
		return nil, "", false
	}

	// backendPath начинается с самого балансировщика — он уже последний в path.
	finalPath := append(path, backendPath[1:]...) //nolint:gocritic // appendAssign: path дальше не меняется, общий массив безопасен

	deliveryStart := r.timestamp
	deliveryProcessing := r.processingDelay(1, 3)
	r.advance(backendLatency + deliveryProcessing)
	r.addLatencyStage("backend_delivery", "Backend delivery", r.timestamp-deliveryStart, map[string]any{
		"path":            backendPath,
		"oneWayLatencyMs": backendLatency,
		"processingMs":    deliveryProcessing,
	})

	return finalPath, backend.ID, true
}
