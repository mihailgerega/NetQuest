package engine

import "github.com/netquest/netquest/backend/internal/model"

// runPing — сценарий icmp_ping: echo request до цели и echo reply обратно.
// Firewall и порты в ping не участвуют — только маршрут и потери пакетов.
//
// Цель ищется по ID узла, IP, hostname или имени (findNode).
// Задержка ответа — путь туда и обратно (RTT = 2 × задержка пути) плюс
// поиск маршрута, а при упавших узлах — ещё и накладные расходы failover.
func (r *runner) runPing(failover bool) {
	source := r.req.Scenario.SourceNodeID
	target := r.req.Scenario.Target

	targetNode, ok := r.findNode(target)
	if !ok {
		r.fail("ping destination not found: " + target)
		return
	}

	path, latency, ok := r.routeOrFail(source, targetNode.ID)
	if !ok {
		return
	}

	r.failoverIfNeeded(failover, source, targetNode.ID, path)
	r.selectRoute(source, targetNode.ID, path, latency)

	if !r.survivePacketLoss(source, targetNode.ID, path, latency, packetLossMessages{
		firstDrop:  "packet dropped by deterministic packet loss",
		secondDrop: "packet dropped by deterministic packet loss",
	}) {
		return
	}

	r.summary.Path = path
	r.summary.Decisions = append(r.summary.Decisions, "Graph route selected for ICMP ping")

	rttStart := r.timestamp
	r.advance(latency * 2)
	r.addLatencyStage("icmp_rtt", "ICMP RTT", r.timestamp-rttStart, map[string]any{
		"path":            path,
		"oneWayLatencyMs": latency,
		"rttMs":           latency * 2,
	})

	r.summary.LatencyFormula = "Ping RTT ≈ one-way path latency × 2 + route lookup + processing delays"
	r.summary.TotalLatencyMs = r.timestamp

	r.emit(model.EventPacketDelivered, model.EventSeverityInfo, "ICMP echo reply delivered", targetNode.ID, source, r.packetID, map[string]any{
		"rttMs": r.summary.TotalLatencyMs,
		"path":  path,
	})
	r.complete("ICMP ping completed")
}
