package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/netquest/netquest/backend/internal/topology"
	"github.com/netquest/netquest/backend/pkg/idgen"
)

const maxSimulationEvents = 10000

const (
	httpsProtocol = "tcp"
	httpsPort     = 443
)

type SimulationEngine interface {
	Run(ctx context.Context, req RunRequest) (RunResult, error)
}

type BasicEngine struct {
	validator topology.Validator
}

type TopologyInvalidError struct {
	Validation topology.ValidationResult
}

func (e TopologyInvalidError) Error() string {
	return "topology is invalid"
}

func NewBasicEngine(validator topology.Validator) *BasicEngine {
	return &BasicEngine{validator: validator}
}

func (e *BasicEngine) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	if req.Seed == 0 {
		req.Seed = 1
	}
	if req.SimulationID == "" {
		return RunResult{}, fmt.Errorf("simulation id is required")
	}

	validation := e.validator.ValidateRaw(req.Topology)
	if !validation.Valid {
		return RunResult{Status: StatusFailed, Seed: req.Seed}, TopologyInvalidError{Validation: validation}
	}

	var doc topology.Document
	if err := json.Unmarshal(req.Topology, &doc); err != nil {
		return RunResult{}, fmt.Errorf("decode topology: %w", err)
	}

	runner := newRunner(req, doc)
	runner.emit(EventSimulationStarted, SeverityInfo, "simulation started", "", "", "", map[string]any{"seed": req.Seed})
	runner.advance(runner.processingDelay(1, 3))
	runner.emit(EventTopologyValidated, SeverityInfo, "topology validated", "", "", "", map[string]any{"valid": true})
	if !runner.validateSourceClient() {
		runner.summary.Status = runner.status
		return RunResult{Status: runner.status, Seed: req.Seed, Events: runner.events, Summary: runner.summary}, nil
	}
	runner.advance(1)
	runner.createPacket()

	switch req.Scenario.Type {
	case "dns_lookup":
		runner.runDNSLookup()
	case "icmp_ping":
		runner.runPing(false)
	case "https_request":
		runner.runHTTPS(false)
	case "failover_demo":
		runner.runHTTPS(true)
	default:
		runner.fail("unsupported scenario type: " + req.Scenario.Type)
	}

	select {
	case <-ctx.Done():
		runner.fail(ctx.Err().Error())
	default:
	}

	if len(runner.events) > maxSimulationEvents {
		runner.events = runner.events[:maxSimulationEvents]
		runner.fail("simulation event limit exceeded")
	}

	runner.summary.Status = runner.status
	runner.summary.ProtocolDetails = runner.buildProtocolDetails()
	return RunResult{Status: runner.status, Seed: req.Seed, Events: runner.events, Summary: runner.summary}, nil
}

type runner struct {
	req       RunRequest
	doc       topology.Document
	nodes     map[string]topology.Node
	rng       *rand.Rand
	events    []Event
	timestamp int64
	packetID  string
	status    Status
	summary   Summary
	routeInfo string
}

func newRunner(req RunRequest, doc topology.Document) *runner {
	nodes := make(map[string]topology.Node, len(doc.Nodes))
	for _, node := range doc.Nodes {
		nodes[node.ID] = node
	}
	return &runner{
		req:    req,
		doc:    doc,
		nodes:  nodes,
		rng:    rand.New(rand.NewSource(req.Seed)),
		status: StatusCompleted,
		summary: Summary{
			Scenario:     req.Scenario.Type,
			Status:       StatusCompleted,
			Seed:         req.Seed,
			Source:       req.Scenario.SourceNodeID,
			SourceNodeID: req.Scenario.SourceNodeID,
			Destination:  req.Scenario.Target,
			Decisions:    []string{},
			Errors:       []string{},
			Path:         []string{},
		},
	}
}

func (r *runner) validateSourceClient() bool {
	sourceID := strings.TrimSpace(r.req.Scenario.SourceNodeID)
	if sourceID == "" {
		r.fail("sourceNodeId is required")
		return false
	}
	source, ok := r.nodes[sourceID]
	if !ok {
		r.fail("source node does not exist")
		return false
	}
	if source.Type != topology.NodeTypeClient {
		r.fail("source node must be a client")
		return false
	}
	if nodeDown(source) {
		r.fail("source client is down")
		return false
	}
	r.summary.Source = nodeName(source)
	r.summary.SourceNodeID = source.ID
	return true
}

func (r *runner) createPacket() {
	r.packetID = fmt.Sprintf("pkt_%016x", r.rng.Uint64())
	r.summary.PacketID = r.packetID
	r.emit(EventPacketCreated, SeverityInfo, "packet created", r.req.Scenario.SourceNodeID, r.req.Scenario.Target, r.packetID, map[string]any{
		"scenario": r.req.Scenario.Type,
	})
	r.advance(1)
}

func (r *runner) runDNSLookup() {
	hostname := targetHost(r.req.Scenario.Target)
	resolved, dnsNode, ttl, dnsPath, _, ok := r.resolveDNS(hostname)
	if !ok {
		return
	}
	r.summary.ResolvedIP = resolved
	r.summary.Path = dnsPath
	if len(r.summary.Path) == 0 {
		r.summary.Path = []string{r.req.Scenario.SourceNodeID, dnsNode.ID}
	}
	r.summary.LatencyFormula = "DNS Lookup latency ≈ path RTT + DNS processing delay"
	r.summary.Decisions = append(r.summary.Decisions, "DNS resolved "+hostname+" to "+resolved)
	r.summary.TotalLatencyMs = r.timestamp
	r.complete("DNS lookup completed")
	_ = ttl
}

func (r *runner) runPing(failover bool) {
	source := r.req.Scenario.SourceNodeID
	target := r.req.Scenario.Target
	targetNode, ok := r.findNode(target)
	if !ok {
		r.fail("ping destination not found: " + target)
		return
	}
	path, latency, ok := r.findPath(source, targetNode.ID)
	if !ok {
		r.emit(EventRouteNotFound, SeverityError, "route not found", source, targetNode.ID, r.packetID, map[string]any{"source": source, "target": targetNode.ID, "explanation": r.routeInfo})
		r.fail("no route from " + source + " to " + targetNode.ID)
		return
	}
	if failover || r.hasDownInfrastructure() {
		r.summary.Failover = true
		r.emit(EventFailoverTriggered, SeverityWarn, "failover triggered", source, targetNode.ID, r.packetID, map[string]any{"reason": "down node or link detected"})
		start := r.timestamp
		delay := r.failoverDelay()
		r.advance(delay)
		r.addLatencyStage("failover_overhead", "Failover overhead", r.timestamp-start, map[string]any{"processingMs": delay})
		r.emit(EventFailoverRouteChanged, SeverityInfo, "route recomputed after failure", source, targetNode.ID, r.packetID, map[string]any{"path": path})
	}
	routeStart := r.timestamp
	routeProcessing := r.processingDelay(1, 3)
	r.advance(routeProcessing)
	r.emit(EventRouteSelected, SeverityInfo, "route selected", source, targetNode.ID, r.packetID, map[string]any{"path": path, "latencyMs": latency, "algorithm": r.routingAlgorithm(source, targetNode.ID), "explanation": r.routeInfo})
	r.addLatencyStage("route_lookup", "Route lookup", r.timestamp-routeStart, map[string]any{"path": path, "oneWayLatencyMs": latency, "processingMs": routeProcessing})
	if r.packetLostOnPath(path) {
		lossStart := r.timestamp
		r.advance(latency)
		r.emit(EventPacketDropped, SeverityWarn, "packet dropped by deterministic packet loss", source, targetNode.ID, r.packetID, map[string]any{"path": path, "attempt": 1})
		retry := r.retryDelay()
		r.advance(retry)
		if r.packetLostOnPath(path) {
			r.advance(latency)
			r.addLatencyStage("packet_loss_retry", "Packet loss/retry", r.timestamp-lossStart, map[string]any{"attempts": 2, "retryDelayMs": retry, "path": path})
			r.emit(EventPacketDropped, SeverityWarn, "packet dropped by deterministic packet loss", source, targetNode.ID, r.packetID, map[string]any{"path": path, "attempt": 2})
			r.fail("packet dropped because link packetLossPercent matched deterministic seed")
			return
		}
		r.addLatencyStage("packet_loss_retry", "Packet loss/retry", r.timestamp-lossStart, map[string]any{"attempts": 2, "retryDelayMs": retry, "path": path})
		r.emit(EventPacketForwarded, SeverityInfo, "packet retry succeeded", source, targetNode.ID, r.packetID, map[string]any{"attempt": 2, "path": path})
	}
	r.summary.Path = path
	r.summary.Decisions = append(r.summary.Decisions, "Graph route selected for ICMP ping")
	rttStart := r.timestamp
	r.advance(latency * 2)
	r.addLatencyStage("icmp_rtt", "ICMP RTT", r.timestamp-rttStart, map[string]any{"path": path, "oneWayLatencyMs": latency, "rttMs": latency * 2})
	r.summary.LatencyFormula = "Ping RTT ≈ one-way path latency × 2 + route lookup + processing delays"
	r.summary.TotalLatencyMs = r.timestamp
	r.emit(EventPacketDelivered, SeverityInfo, "ICMP echo reply delivered", targetNode.ID, source, r.packetID, map[string]any{"rttMs": r.summary.TotalLatencyMs, "path": path})
	r.complete("ICMP ping completed")
}

func (r *runner) runHTTPS(failover bool) {
	source := r.req.Scenario.SourceNodeID
	hostname := targetHost(r.req.Scenario.Target)
	resolvedIP := hostname
	if net.ParseIP(hostname) == nil {
		var ok bool
		resolvedIP, _, _, _, _, ok = r.resolveDNS(hostname)
		if !ok {
			r.fail("DNS NXDOMAIN for " + hostname)
			return
		}
		r.summary.ResolvedIP = resolvedIP
	} else {
		r.summary.Decisions = append(r.summary.Decisions, "Target is already an IP address: "+hostname)
		r.summary.ResolvedIP = resolvedIP
	}
	destination, ok := r.findNodeByIP(resolvedIP)
	if !ok {
		r.fail("resolved IP does not belong to a topology node: " + resolvedIP)
		return
	}

	path, latency, ok := r.findPath(source, destination.ID)
	if !ok {
		r.emit(EventRouteNotFound, SeverityError, "route not found", source, destination.ID, r.packetID, map[string]any{"source": source, "target": destination.ID, "explanation": r.routeInfo})
		r.fail("no route from " + source + " to " + destination.ID)
		return
	}
	if failover || r.hasDownInfrastructure() {
		r.summary.Failover = true
		r.emit(EventFailoverTriggered, SeverityWarn, "failover triggered", source, destination.ID, r.packetID, map[string]any{"reason": "down node or link detected"})
		start := r.timestamp
		delay := r.failoverDelay()
		r.advance(delay)
		r.addLatencyStage("failover_overhead", "Failover overhead", r.timestamp-start, map[string]any{"processingMs": delay})
		r.emit(EventFailoverRouteChanged, SeverityInfo, "route recomputed after failure", source, destination.ID, r.packetID, map[string]any{"path": path})
	}
	routeStart := r.timestamp
	routeProcessing := r.processingDelay(1, 3)
	r.advance(routeProcessing)
	r.emit(EventRouteSelected, SeverityInfo, "route selected", source, destination.ID, r.packetID, map[string]any{"path": path, "latencyMs": latency, "algorithm": r.routingAlgorithm(source, destination.ID), "explanation": r.routeInfo})
	r.addLatencyStage("route_lookup", "Route lookup", r.timestamp-routeStart, map[string]any{"path": path, "oneWayLatencyMs": latency, "processingMs": routeProcessing})
	r.summary.Decisions = append(r.summary.Decisions, "Graph route selected to "+destination.ID)

	if r.packetLostOnPath(path) {
		lossStart := r.timestamp
		r.advance(latency)
		r.emit(EventPacketDropped, SeverityWarn, "packet dropped before TCP handshake", source, destination.ID, r.packetID, map[string]any{"path": path, "attempt": 1})
		retry := r.retryDelay()
		r.advance(retry)
		if r.packetLostOnPath(path) {
			r.advance(latency)
			r.addLatencyStage("packet_loss_retry", "Packet loss/retry", r.timestamp-lossStart, map[string]any{"attempts": 2, "retryDelayMs": retry, "path": path})
			r.emit(EventPacketDropped, SeverityWarn, "packet retry dropped before TCP handshake", source, destination.ID, r.packetID, map[string]any{"path": path, "attempt": 2})
			r.fail("packet dropped because link packetLossPercent matched deterministic seed")
			return
		}
		r.addLatencyStage("packet_loss_retry", "Packet loss/retry", r.timestamp-lossStart, map[string]any{"attempts": 2, "retryDelayMs": retry, "path": path})
		r.emit(EventPacketForwarded, SeverityInfo, "packet retry succeeded", source, destination.ID, r.packetID, map[string]any{"attempt": 2, "path": path})
	}

	if !r.firewallAllows(path, resolvedIP, httpsPort) {
		return
	}
	if destination.Type == topology.NodeTypeServer && !r.ensureServerPortOpen(destination, httpsProtocol, httpsPort, source, destination.ID) {
		return
	}

	tcpStart := r.timestamp
	r.emit(EventTCPHandshakeStart, SeverityInfo, "TCP handshake started", source, destination.ID, r.packetID, nil)
	r.emit(EventTCPSYN, SeverityInfo, "TCP SYN sent", source, destination.ID, r.packetID, nil)
	r.advance(latency)
	r.emit(EventTCPSYNACK, SeverityInfo, "TCP SYN-ACK received", destination.ID, source, r.packetID, nil)
	r.advance(latency)
	r.emit(EventTCPACK, SeverityInfo, "TCP ACK sent", source, destination.ID, r.packetID, nil)
	tcpProcessing := r.processingDelay(1, 2)
	r.advance(tcpProcessing)
	r.emit(EventTCPHandshakeDone, SeverityInfo, "TCP handshake completed", source, destination.ID, r.packetID, nil)
	r.addLatencyStage("tcp_handshake", "TCP Handshake", r.timestamp-tcpStart, map[string]any{"path": path, "oneWayLatencyMs": latency, "rttMs": latency * 2, "processingMs": tcpProcessing})
	tlsStart := r.timestamp
	tlsSetupProcessing := r.processingDelay(1, 3)
	r.advance(tlsSetupProcessing)
	r.emit(EventTLSHandshakeStart, SeverityInfo, "TLS handshake started", source, destination.ID, r.packetID, nil)
	r.emit(EventTLSClientHello, SeverityInfo, "TLS ClientHello sent", source, destination.ID, r.packetID, map[string]any{"serverName": hostname})
	r.advance(latency)
	r.emit(EventTLSServerHello, SeverityInfo, "TLS ServerHello received", destination.ID, source, r.packetID, nil)
	tlsCertificateProcessing := r.processingDelay(2, 5)
	r.advance(tlsCertificateProcessing)
	r.emit(EventTLSCertValidated, SeverityInfo, "TLS certificate validated", destination.ID, source, r.packetID, map[string]any{"hostname": hostname})
	r.advance(latency)
	r.emit(EventTLSHandshakeDone, SeverityInfo, "TLS handshake completed", source, destination.ID, r.packetID, nil)
	r.addLatencyStage("tls_handshake", "TLS Handshake", r.timestamp-tlsStart, map[string]any{"path": path, "oneWayLatencyMs": latency, "roundTrips": 1, "rttMs": latency * 2, "processingMs": tlsSetupProcessing + tlsCertificateProcessing})

	finalPath := path
	selectedBackend := ""
	if destination.Type == topology.NodeTypeLoadBalancer {
		lbStart := r.timestamp
		lbProcessing := r.processingDelay(1, 5)
		r.advance(lbProcessing)
		backend, ok := r.selectBackend(destination, httpsPort)
		if !ok {
			return
		}
		selectedBackend = backend.ID
		r.summary.SelectedBackend = selectedBackend
		r.summary.SelectedBackendNodeID = backend.ID
		r.summary.SelectedBackendName = nodeName(backend)
		r.addLatencyStage("load_balancer_decision", "Load Balancer decision", r.timestamp-lbStart, map[string]any{"processingMs": lbProcessing, "selectedBackendNodeId": backend.ID, "selectedBackendName": nodeName(backend), "healthyBackends": r.summary.HealthyBackends, "skippedBackends": r.summary.SkippedBackends})
		backendPath, backendLatency, ok := r.findPath(destination.ID, backend.ID)
		if !ok {
			r.fail("Load balancer has no healthy backends available.")
			return
		}
		finalPath = append(path, backendPath[1:]...)
		latency += backendLatency
		deliveryStart := r.timestamp
		deliveryProcessing := r.processingDelay(1, 3)
		r.advance(backendLatency + deliveryProcessing)
		r.addLatencyStage("backend_delivery", "Backend delivery", r.timestamp-deliveryStart, map[string]any{"path": backendPath, "oneWayLatencyMs": backendLatency, "processingMs": deliveryProcessing})
	} else {
		deliveryStart := r.timestamp
		deliveryProcessing := r.processingDelay(1, 3)
		r.advance(latency + deliveryProcessing)
		r.addLatencyStage("packet_delivery", "Packet delivery", r.timestamp-deliveryStart, map[string]any{"path": path, "oneWayLatencyMs": latency, "processingMs": deliveryProcessing})
	}

	r.summary.Path = finalPath
	r.summary.LatencyFormula = "HTTPS latency ≈ DNS + route lookup + firewall decision + TCP handshake + TLS handshake + Load Balancer decision + delivery"
	r.summary.TotalLatencyMs = r.timestamp
	if selectedBackend != "" {
		r.summary.Decisions = append(r.summary.Decisions, "Load balancer selected "+r.summary.SelectedBackendName)
	}
	r.emit(EventPacketDelivered, SeverityInfo, "HTTPS request delivered", source, destination.ID, r.packetID, map[string]any{
		"path":            finalPath,
		"selectedBackend": selectedBackend,
		"latencyMs":       r.summary.TotalLatencyMs,
	})
	r.complete("HTTPS request completed")
}

func (r *runner) resolveDNS(hostname string) (string, topology.Node, int, []string, int64, bool) {
	source := r.req.Scenario.SourceNodeID
	dns := topology.Node{}
	for _, node := range r.doc.Nodes {
		if node.Type == topology.NodeTypeDNS && !nodeDown(node) {
			dns = node
			break
		}
	}
	if dns.ID == "" {
		r.emit(EventDNSError, SeverityError, "DNS resolver is not available", source, "", r.packetID, map[string]any{"hostname": hostname})
		r.fail("DNS resolver is not available")
		return "", topology.Node{}, 0, nil, 0, false
	}
	path, latency, ok := r.findPath(source, dns.ID)
	if !ok {
		r.emit(EventRouteNotFound, SeverityError, "route to DNS resolver not found", source, dns.ID, r.packetID, map[string]any{"source": source, "target": dns.ID, "explanation": r.routeInfo})
		r.fail("no route from " + source + " to DNS resolver " + dns.ID)
		return "", dns, 0, nil, 0, false
	}
	start := r.timestamp
	processing := r.processingDelay(1, 4)
	r.emit(EventDNSQuery, SeverityInfo, "DNS query", source, dns.ID, r.packetID, map[string]any{"hostname": hostname, "type": "A", "path": path, "oneWayLatencyMs": latency})
	r.advance(latency + processing + latency)
	r.addLatencyStage("dns_lookup", "DNS Lookup", r.timestamp-start, map[string]any{"hostname": hostname, "path": path, "oneWayLatencyMs": latency, "processingMs": processing, "rttMs": latency * 2})
	for _, record := range anySlice(dns.Config["records"]) {
		m := anyMap(record)
		if strings.EqualFold(stringValue(m["name"]), hostname) && strings.EqualFold(defaultString(m["type"], "A"), "A") {
			value := stringValue(m["value"])
			ttl := intValue(m["ttl"], 300)
			r.emit(EventDNSResponse, SeverityInfo, "DNS response", dns.ID, source, r.packetID, map[string]any{"hostname": hostname, "value": value, "ttl": ttl, "latencyMs": latency * 2})
			return value, dns, ttl, path, latency, true
		}
	}
	r.emit(EventDNSError, SeverityError, "DNS NXDOMAIN", dns.ID, source, r.packetID, map[string]any{"hostname": hostname, "rcode": "NXDOMAIN"})
	r.fail("DNS NXDOMAIN for " + hostname)
	return "", dns, 0, path, latency, false
}

func (r *runner) firewallAllows(path []string, destinationIP string, port int) bool {
	sourceIP := ""
	if source, ok := r.nodes[r.req.Scenario.SourceNodeID]; ok {
		sourceIP = nodeIP(source)
	}
	for _, nodeID := range path {
		node := r.nodes[nodeID]
		if node.Type != topology.NodeTypeFirewall {
			continue
		}
		start := r.timestamp
		processing := r.processingDelay(1, 5)
		r.advance(processing)
		allowed, decision := evaluateFirewall(node, sourceIP, destinationIP, "tcp", port)
		eventType := EventFirewallDecision
		severity := SeverityInfo
		if !allowed {
			eventType = EventFirewallDenied
			severity = SeverityWarn
		}
		r.emit(eventType, severity, decision, r.req.Scenario.SourceNodeID, node.ID, r.packetID, map[string]any{"decision": decision, "allowed": allowed})
		r.addLatencyStage("firewall_decision", "Firewall decision", r.timestamp-start, map[string]any{"processingMs": processing, "allowed": allowed, "decision": decision, "nodeId": node.ID})
		r.summary.Decisions = append(r.summary.Decisions, decision)
		if !allowed {
			r.emit(EventPacketDropped, SeverityWarn, "packet dropped by firewall", r.req.Scenario.SourceNodeID, node.ID, r.packetID, nil)
			r.fail(decision)
			return false
		}
		return true
	}
	r.summary.Decisions = append(r.summary.Decisions, "No firewall on path")
	return true
}

func (r *runner) selectBackend(lb topology.Node, port int) (topology.Node, bool) {
	candidates := r.lbBackendCandidates(lb)
	if len(candidates) == 0 {
		r.emit(EventLBBackendUnhealthy, SeverityError, "load balancer has no backend pool", lb.ID, "", r.packetID, map[string]any{"healthyBackends": []string{}, "skippedBackends": []BackendSkip{}})
		r.fail("Load balancer has no healthy backends available.")
		return topology.Node{}, false
	}

	healthy := make([]topology.Node, 0, len(candidates))
	healthyIDs := make([]string, 0, len(candidates))
	skipped := make([]BackendSkip, 0)
	for _, candidate := range candidates {
		nodeID := candidate.NodeID
		if nodeID == "" {
			skipped = append(skipped, BackendSkip{Reason: "backend nodeId is empty"})
			continue
		}
		node, ok := r.nodes[nodeID]
		if !ok {
			skipped = append(skipped, BackendSkip{NodeID: nodeID, Reason: "backend node does not exist"})
			continue
		}
		if node.Type != topology.NodeTypeServer {
			skipped = append(skipped, BackendSkip{NodeID: nodeID, Name: nodeName(node), Reason: "backend node is not a server"})
			continue
		}
		if !candidate.Enabled {
			skipped = append(skipped, BackendSkip{NodeID: nodeID, Name: nodeName(node), Reason: "backend is disabled"})
			continue
		}
		if nodeDown(node) {
			skipped = append(skipped, BackendSkip{NodeID: nodeID, Name: nodeName(node), Reason: "node is down"})
			continue
		}
		portCheck := serverPortCheck(node, httpsProtocol, port)
		if !portCheck.Open {
			skipped = append(skipped, BackendSkip{NodeID: nodeID, Name: nodeName(node), Reason: fmt.Sprintf("server port %s/%d is closed", httpsProtocol, port)})
			continue
		}
		if _, _, ok := r.findPath(lb.ID, nodeID); !ok {
			skipped = append(skipped, BackendSkip{NodeID: nodeID, Name: nodeName(node), Reason: "no active path from load balancer"})
			continue
		}
		healthy = append(healthy, node)
		healthyIDs = append(healthyIDs, nodeID)
	}

	r.summary.HealthyBackends = healthyIDs
	r.summary.SkippedBackends = skipped
	if len(skipped) > 0 {
		r.emit(EventLBBackendUnhealthy, SeverityWarn, "load balancer skipped unavailable backend(s)", lb.ID, "", r.packetID, map[string]any{
			"healthyBackends": healthyIDs,
			"skippedBackends": skipped,
		})
	}
	if len(healthy) == 0 {
		r.emit(EventLBBackendUnhealthy, SeverityError, "load balancer has no healthy backends available", lb.ID, "", r.packetID, map[string]any{
			"healthyBackends": healthyIDs,
			"skippedBackends": skipped,
		})
		r.fail("Load balancer has no healthy backends available.")
		return topology.Node{}, false
	}

	sort.Slice(healthy, func(i, j int) bool { return healthy[i].ID < healthy[j].ID })
	sort.Strings(healthyIDs)
	algorithm := defaultString(lb.Config["algorithm"], "round_robin")
	selected := healthy[0]
	switch algorithm {
	case "round_robin":
		selected = healthy[int(r.req.Seed%int64(len(healthy)))]
	case "least_connections":
		sort.SliceStable(healthy, func(i, j int) bool {
			left := intValue(healthy[i].Config["activeConnections"], 0)
			right := intValue(healthy[j].Config["activeConnections"], 0)
			if left == right {
				return healthy[i].ID < healthy[j].ID
			}
			return left < right
		})
		selected = healthy[0]
	}

	reason := fmt.Sprintf("%s selected %s", algorithm, nodeName(selected))
	if len(skipped) > 0 {
		reason = fmt.Sprintf("%s; skipped %d unavailable backend(s)", reason, len(skipped))
	}
	r.emit(EventLBBackendSelected, SeverityInfo, "load balancer selected backend", lb.ID, selected.ID, r.packetID, map[string]any{
		"algorithm":             algorithm,
		"selectedBackendNodeId": selected.ID,
		"selectedBackendName":   nodeName(selected),
		"reason":                reason,
		"healthyBackends":       healthyIDs,
		"skippedBackends":       skipped,
	})
	r.ensureServerPortOpen(selected, httpsProtocol, port, lb.ID, selected.ID)
	r.summary.Decisions = append(r.summary.Decisions, "Load balancer selected "+nodeName(selected)+": "+reason)
	return selected, true
}

type lbBackendCandidate struct {
	NodeID         string
	Enabled        bool
	AutoDiscovered bool
}

func (r *runner) lbBackendCandidates(lb topology.Node) []lbBackendCandidate {
	candidates := make([]lbBackendCandidate, 0)
	seen := map[string]struct{}{}
	for _, backend := range anySlice(lb.Config["backends"]) {
		m := anyMap(backend)
		nodeID := stringValue(m["nodeId"])
		if nodeID != "" {
			if _, ok := seen[nodeID]; ok {
				continue
			}
			seen[nodeID] = struct{}{}
		}
		candidates = append(candidates, lbBackendCandidate{
			NodeID:  nodeID,
			Enabled: boolValue(m["enabled"], boolValue(m["healthy"], true)),
		})
	}

	if !boolValue(lb.Config["autoDiscoverConnectedServers"], false) {
		return candidates
	}
	for _, link := range r.doc.Links {
		if link.SourceNodeID != lb.ID && link.TargetNodeID != lb.ID {
			continue
		}
		otherID := link.TargetNodeID
		if otherID == lb.ID {
			otherID = link.SourceNodeID
		}
		if _, ok := seen[otherID]; ok {
			continue
		}
		node, ok := r.nodes[otherID]
		if !ok || node.Type != topology.NodeTypeServer {
			continue
		}
		seen[otherID] = struct{}{}
		candidates = append(candidates, lbBackendCandidate{NodeID: otherID, Enabled: true, AutoDiscovered: true})
		r.emit(EventLBBackendDiscovered, SeverityInfo, "load balancer discovered connected server", lb.ID, otherID, r.packetID, map[string]any{
			"backend": otherID,
			"name":    nodeName(node),
		})
	}
	return candidates
}

func (r *runner) findPath(sourceID, targetID string) ([]string, int64, bool) {
	r.routeInfo = ""
	if r.usesRoutingTables(sourceID, targetID) {
		path, latency, ok := r.findRoutedPath(sourceID, targetID)
		if ok {
			return path, latency, true
		}
		if r.routeInfo == "" {
			r.routeInfo = "Route table mode is enabled, but no matching route could be built."
		}
		return nil, 0, false
	}
	path, latency, ok := r.findGraphPath(sourceID, targetID)
	if ok {
		r.routeInfo = "Selected lowest-latency active graph path."
	}
	return path, latency, ok
}

func (r *runner) findGraphPath(sourceID, targetID string) ([]string, int64, bool) {
	if sourceID == "" || targetID == "" {
		return nil, 0, false
	}
	if sourceID == targetID {
		return []string{sourceID}, 0, true
	}
	if nodeDown(r.nodes[sourceID]) || nodeDown(r.nodes[targetID]) {
		return nil, 0, false
	}

	type edge struct {
		to      string
		latency int64
	}
	graph := map[string][]edge{}
	for _, link := range r.doc.Links {
		if linkDown(link) || nodeDown(r.nodes[link.SourceNodeID]) || nodeDown(r.nodes[link.TargetNodeID]) {
			continue
		}
		latency := int64(intValue(link.Config["latencyMs"], intValue(link.Config["cost"], 10)))
		graph[link.SourceNodeID] = append(graph[link.SourceNodeID], edge{to: link.TargetNodeID, latency: latency})
		graph[link.TargetNodeID] = append(graph[link.TargetNodeID], edge{to: link.SourceNodeID, latency: latency})
	}

	dist := map[string]int64{sourceID: 0}
	prev := map[string]string{}
	visited := map[string]bool{}
	for {
		current := ""
		best := int64(math.MaxInt64)
		for nodeID, value := range dist {
			if !visited[nodeID] && value < best {
				current = nodeID
				best = value
			}
		}
		if current == "" {
			break
		}
		if current == targetID {
			break
		}
		visited[current] = true
		for _, next := range graph[current] {
			candidate := dist[current] + next.latency
			if existing, ok := dist[next.to]; !ok || candidate < existing {
				dist[next.to] = candidate
				prev[next.to] = current
			}
		}
	}
	total, ok := dist[targetID]
	if !ok {
		return nil, 0, false
	}
	path := []string{targetID}
	for current := targetID; current != sourceID; {
		parent, ok := prev[current]
		if !ok {
			return nil, 0, false
		}
		path = append(path, parent)
		current = parent
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path, total, true
}

func (r *runner) findDirectPath(sourceID, targetID string) ([]string, int64, bool) {
	if sourceID == "" || targetID == "" {
		return nil, 0, false
	}
	if sourceID == targetID {
		return []string{sourceID}, 0, true
	}
	if nodeDown(r.nodes[sourceID]) || nodeDown(r.nodes[targetID]) {
		return nil, 0, false
	}
	for _, link := range r.doc.Links {
		if linkDown(link) {
			continue
		}
		if (link.SourceNodeID == sourceID && link.TargetNodeID == targetID) || (link.SourceNodeID == targetID && link.TargetNodeID == sourceID) {
			return []string{sourceID, targetID}, int64(intValue(link.Config["latencyMs"], intValue(link.Config["cost"], 10))), true
		}
	}
	return nil, 0, false
}

func (r *runner) usesRoutingTables(sourceID, targetID string) bool {
	source, sourceOK := r.nodes[sourceID]
	target, targetOK := r.nodes[targetID]
	if !sourceOK || !targetOK {
		return false
	}
	if hasRoutingConfig(source) || hasRoutingConfig(target) {
		return true
	}
	for _, node := range r.doc.Nodes {
		if node.Type == topology.NodeTypeRouter && len(anySlice(node.Config["routes"])) > 0 {
			return true
		}
	}
	return false
}

func (r *runner) routingAlgorithm(sourceID, targetID string) string {
	if r.usesRoutingTables(sourceID, targetID) {
		return "route_table"
	}
	return "graph_path"
}

func (r *runner) findRoutedPath(sourceID, targetID string) ([]string, int64, bool) {
	source, sourceOK := r.nodes[sourceID]
	target, targetOK := r.nodes[targetID]
	if !sourceOK || !targetOK {
		r.routeInfo = "source or target node does not exist"
		return nil, 0, false
	}
	if nodeDown(source) || nodeDown(target) {
		r.routeInfo = "source or target node is down"
		return nil, 0, false
	}
	destinationIP := nodeIP(target)
	if destinationIP == "" {
		r.routeInfo = "target node has no IP address"
		return nil, 0, false
	}
	if sameSubnet(source, destinationIP) {
		path, latency, ok := r.findGraphPath(sourceID, targetID)
		if ok {
			r.routeInfo = "Destination is in the source subnet; direct route selected."
		}
		return path, latency, ok
	}

	fullPath := []string{sourceID}
	totalLatency := int64(0)
	current := source
	if source.Type != topology.NodeTypeRouter {
		gatewayIP := stringValue(source.Config["defaultGateway"])
		if gatewayIP == "" {
			r.routeInfo = "source client has no defaultGateway for off-subnet destination"
			return nil, 0, false
		}
		gateway, ok := r.findNodeByAnyIP(gatewayIP)
		if !ok {
			r.routeInfo = "defaultGateway " + gatewayIP + " does not match any node interface"
			return nil, 0, false
		}
		path, latency, ok := r.findDirectPath(source.ID, gateway.ID)
		if !ok {
			r.routeInfo = "defaultGateway is not reachable through active links"
			return nil, 0, false
		}
		fullPath = appendPath(fullPath, path)
		totalLatency += latency
		current = gateway
	}

	visited := map[string]bool{sourceID: true}
	for hops := 0; hops < len(r.doc.Nodes)+2; hops++ {
		if current.ID == targetID {
			r.routeInfo = "Route table reached destination."
			return fullPath, totalLatency, true
		}
		if sameSubnet(current, destinationIP) {
			path, latency, ok := r.findGraphPath(current.ID, targetID)
			if ok {
				fullPath = appendPath(fullPath, path)
				totalLatency += latency
				r.routeInfo = "Router has directly connected subnet for destination."
				return fullPath, totalLatency, true
			}
		}
		routes := routeCandidates(current, destinationIP)
		if len(routes) == 0 {
			r.routeInfo = "router " + current.ID + " has no route matching " + destinationIP
			return nil, 0, false
		}
		selected := false
		lastReason := ""
		for _, route := range routes {
			next := target
			if route.gateway != "" {
				var found bool
				next, found = r.findNodeByAnyIP(route.gateway)
				if !found {
					lastReason = "route gateway " + route.gateway + " does not match any node interface"
					continue
				}
			}
			if visited[next.ID] {
				lastReason = "route table loop detected at " + next.ID
				continue
			}
			path, latency, ok := r.findDirectPath(current.ID, next.ID)
			if !ok {
				lastReason = "next hop " + next.ID + " is not reachable through active links"
				continue
			}
			fullPath = appendPath(fullPath, path)
			totalLatency += latency
			visited[next.ID] = true
			current = next
			selected = true
			r.routeInfo = fmt.Sprintf("Route table selected %s via %s metric %d.", route.destination, defaultString(route.gateway, "direct"), route.metric)
			break
		}
		if !selected {
			if lastReason == "" {
				lastReason = "no reachable next hop for matching route"
			}
			r.routeInfo = lastReason
			return nil, 0, false
		}
	}
	r.routeInfo = "route lookup exceeded maximum hop count"
	return nil, 0, false
}

type routeEntry struct {
	destination string
	gateway     string
	iface       string
	metric      int
	prefixLen   int
}

func bestRoute(node topology.Node, destinationIP string) (routeEntry, bool) {
	routes := routeCandidates(node, destinationIP)
	if len(routes) == 0 {
		return routeEntry{}, false
	}
	return routes[0], true
}

func routeCandidates(node topology.Node, destinationIP string) []routeEntry {
	routes := anySlice(node.Config["routes"])
	candidates := []routeEntry{}
	for _, item := range routes {
		m := anyMap(item)
		destination := defaultString(m["destination"], "0.0.0.0/0")
		if !cidrContains(destination, destinationIP) {
			continue
		}
		candidates = append(candidates, routeEntry{
			destination: destination,
			gateway:     stringValue(m["gateway"]),
			iface:       stringValue(m["interface"]),
			metric:      intValue(m["metric"], 100),
			prefixLen:   prefixLen(destination),
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].prefixLen != candidates[j].prefixLen {
			return candidates[i].prefixLen > candidates[j].prefixLen
		}
		return candidates[i].metric < candidates[j].metric
	})
	return candidates
}

func (r *runner) findNodeByAnyIP(ip string) (topology.Node, bool) {
	for _, node := range r.doc.Nodes {
		if nodeIP(node) == ip {
			return node, true
		}
		for _, iface := range anySlice(node.Config["interfaces"]) {
			if stringValue(anyMap(iface)["ip"]) == ip {
				return node, true
			}
		}
	}
	return topology.Node{}, false
}

func hasRoutingConfig(node topology.Node) bool {
	return stringValue(node.Config["cidr"]) != "" ||
		stringValue(node.Config["defaultGateway"]) != "" ||
		len(anySlice(node.Config["interfaces"])) > 0 ||
		len(anySlice(node.Config["routes"])) > 0
}

func sameSubnet(node topology.Node, destinationIP string) bool {
	if cidr := stringValue(node.Config["cidr"]); cidr != "" && cidrContains(cidr, destinationIP) {
		return true
	}
	for _, iface := range anySlice(node.Config["interfaces"]) {
		if cidr := stringValue(anyMap(iface)["cidr"]); cidr != "" && cidrContains(cidr, destinationIP) {
			return true
		}
	}
	return false
}

func cidrContains(cidr, ipValue string) bool {
	ip := net.ParseIP(ipValue)
	if ip == nil {
		return false
	}
	if !strings.Contains(cidr, "/") {
		return cidr == ipValue
	}
	_, network, err := net.ParseCIDR(cidr)
	return err == nil && network.Contains(ip)
}

func prefixLen(cidr string) int {
	if !strings.Contains(cidr, "/") {
		return 32
	}
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0
	}
	ones, _ := network.Mask.Size()
	return ones
}

func appendPath(current, segment []string) []string {
	if len(segment) == 0 {
		return current
	}
	start := 0
	if len(current) > 0 && current[len(current)-1] == segment[0] {
		start = 1
	}
	return append(current, segment[start:]...)
}

func (r *runner) packetLostOnPath(path []string) bool {
	for _, link := range r.doc.Links {
		if !linkInPath(link, path) {
			continue
		}
		loss := floatValue(link.Config["packetLossPercent"], 0)
		if loss > 0 && r.rng.Float64()*100 < loss {
			return true
		}
	}
	return false
}

func (r *runner) findNode(target string) (topology.Node, bool) {
	if node, ok := r.nodes[target]; ok {
		return node, true
	}
	for _, node := range r.doc.Nodes {
		if nodeIP(node) == target || stringValue(node.Config["hostname"]) == target || node.Name == target {
			return node, true
		}
	}
	return topology.Node{}, false
}

func (r *runner) findNodeByIP(ip string) (topology.Node, bool) {
	for _, node := range r.doc.Nodes {
		if nodeIP(node) == ip {
			return node, true
		}
	}
	return topology.Node{}, false
}

func (r *runner) hasDownInfrastructure() bool {
	for _, node := range r.doc.Nodes {
		if nodeDown(node) {
			return true
		}
	}
	for _, link := range r.doc.Links {
		if linkDown(link) {
			return true
		}
	}
	return false
}

func (r *runner) complete(message string) {
	if r.status == StatusFailed {
		return
	}
	if r.summary.TotalLatencyMs == 0 {
		r.summary.TotalLatencyMs = r.timestamp
	}
	r.emit(EventSimulationCompleted, SeverityInfo, "simulation completed", "", "", r.packetID, map[string]any{"summary": r.summary, "message": message})
}

func (r *runner) fail(message string) {
	if r.status == StatusFailed {
		return
	}
	r.status = StatusFailed
	r.summary.Status = StatusFailed
	if r.summary.TotalLatencyMs == 0 {
		r.summary.TotalLatencyMs = r.timestamp
	}
	code := errorCodeForMessage(message)
	if code != "" {
		if r.summary.Metadata == nil {
			r.summary.Metadata = map[string]any{}
		}
		r.summary.Metadata["errorCode"] = code
	}
	r.summary.Errors = append(r.summary.Errors, message)
	r.emit(EventSimulationFailed, SeverityError, "simulation failed", "", "", r.packetID, map[string]any{"error": message, "code": code, "summary": r.summary})
}

func (r *runner) emit(eventType EventType, severity Severity, message, source, target, packetID string, details map[string]any) {
	sequence := int64(len(r.events) + 1)
	if details == nil {
		details = map[string]any{}
	}
	r.events = append(r.events, Event{
		ID:             eventID(r.req.SimulationID, sequence),
		SimulationID:   r.req.SimulationID,
		SequenceNumber: sequence,
		Type:           eventType,
		TimestampMs:    r.timestamp,
		SourceNodeID:   source,
		TargetNodeID:   target,
		PacketID:       packetID,
		Severity:       severity,
		Message:        message,
		Details:        details,
	})
}

func (r *runner) buildProtocolDetails() ProtocolDetails {
	details := ProtocolDetails{
		Summary: map[string]any{
			"sourceNodeId":          r.summary.SourceNodeID,
			"destination":           r.summary.Destination,
			"status":                r.summary.Status,
			"totalLatencyMs":        r.summary.TotalLatencyMs,
			"path":                  r.summary.Path,
			"selectedBackendNodeId": r.summary.SelectedBackendNodeID,
			"selectedBackendName":   r.summary.SelectedBackendName,
		},
		Errors: []map[string]any{},
	}

	tcpEvents := []string{}
	tlsEvents := []string{}
	for _, event := range r.events {
		switch event.Type {
		case EventDNSQuery:
			details.DNS = mergeProtocolMap(details.DNS, map[string]any{
				"hostname":   event.Details["hostname"],
				"resolver":   event.TargetNodeID,
				"queryEvent": event.Type,
				"path":       event.Details["path"],
			})
		case EventDNSResponse:
			details.DNS = mergeProtocolMap(details.DNS, map[string]any{
				"responseEvent": event.Type,
				"resolvedIp":    event.Details["value"],
				"ttl":           event.Details["ttl"],
				"recordType":    "A",
			})
		case EventDNSError:
			details.DNS = mergeProtocolMap(details.DNS, map[string]any{"error": event.Message, "details": event.Details})
		case EventRouteSelected:
			details.Routing = map[string]any{
				"sourceNodeId": event.SourceNodeID,
				"targetNodeId": event.TargetNodeID,
				"path":         event.Details["path"],
				"latencyMs":    event.Details["latencyMs"],
				"algorithm":    defaultString(event.Details["algorithm"], "graph_path"),
				"explanation":  defaultString(event.Details["explanation"], "Selected lowest-latency active path."),
			}
		case EventRouteNotFound:
			details.Routing = mergeProtocolMap(details.Routing, map[string]any{
				"error":       event.Message,
				"explanation": defaultString(event.Details["explanation"], "No active route connects source and target."),
				"details":     event.Details,
			})
		case EventFirewallDecision, EventFirewallDenied:
			details.Firewall = map[string]any{
				"nodeId":      event.TargetNodeID,
				"decision":    event.Details["decision"],
				"allowed":     event.Details["allowed"],
				"protocol":    "tcp",
				"port":        443,
				"explanation": event.Message,
			}
		case EventTCPHandshakeStart, EventTCPSYN, EventTCPSYNACK, EventTCPACK, EventTCPHandshakeDone:
			tcpEvents = append(tcpEvents, string(event.Type))
		case EventTLSHandshakeStart, EventTLSClientHello, EventTLSServerHello, EventTLSCertValidated, EventTLSHandshakeDone:
			tlsEvents = append(tlsEvents, string(event.Type))
			if event.Type == EventTLSClientHello {
				details.TLS = mergeProtocolMap(details.TLS, map[string]any{"hostname": event.Details["serverName"]})
			}
		case EventServerPortOpen, EventServerPortClosed:
			details.Server = map[string]any{
				"nodeId":          event.TargetNodeID,
				"nodeName":        event.Details["nodeName"],
				"protocol":        event.Details["protocol"],
				"port":            event.Details["port"],
				"open":            event.Details["open"],
				"implicitOpen":    event.Details["implicitOpen"],
				"openPorts":       event.Details["openPorts"],
				"matchedOpenPort": event.Details["matchedOpenPort"],
				"event":           event.Type,
				"message":         event.Message,
			}
		case EventLBBackendSelected:
			details.LoadBalancer = map[string]any{
				"algorithm":             event.Details["algorithm"],
				"selectedBackendNodeId": event.Details["selectedBackendNodeId"],
				"selectedBackendName":   event.Details["selectedBackendName"],
				"reason":                event.Details["reason"],
				"healthyBackends":       event.Details["healthyBackends"],
				"skippedBackends":       event.Details["skippedBackends"],
			}
		case EventLBBackendUnhealthy:
			details.LoadBalancer = mergeProtocolMap(details.LoadBalancer, map[string]any{
				"unhealthyEvent":  event.Message,
				"healthyBackends": event.Details["healthyBackends"],
				"skippedBackends": event.Details["skippedBackends"],
			})
		case EventSimulationFailed:
			details.Errors = append(details.Errors, map[string]any{
				"code":             event.Details["code"],
				"userMessage":      errorUserMessage(defaultString(event.Details["code"], "SIMULATION_FAILED")),
				"technicalMessage": event.Details["error"],
				"suggestedFix":     errorSuggestedFix(defaultString(event.Details["code"], "SIMULATION_FAILED")),
			})
		}
	}
	if len(tcpEvents) > 0 {
		details.TCP = mergeProtocolMap(details.TCP, map[string]any{
			"events":      tcpEvents,
			"explanation": "TCP handshake uses SYN, SYN-ACK and ACK over the selected route.",
		})
	}
	if len(tlsEvents) > 0 {
		details.TLS = mergeProtocolMap(details.TLS, map[string]any{
			"events":      tlsEvents,
			"validation":  "simulated",
			"explanation": "TLS is virtual; NetQuest does not perform real cryptography.",
		})
	}
	return details
}

func mergeProtocolMap(current map[string]any, updates map[string]any) map[string]any {
	if current == nil {
		current = map[string]any{}
	}
	for key, value := range updates {
		current[key] = value
	}
	return current
}

func (r *runner) addLatencyStage(stage, label string, durationMs int64, details map[string]any) {
	if durationMs < 0 {
		durationMs = 0
	}
	if details == nil {
		details = map[string]any{}
	}
	r.summary.LatencyBreakdown = append(r.summary.LatencyBreakdown, LatencyStage{
		Stage:      stage,
		Label:      label,
		DurationMs: durationMs,
		Details:    details,
	})
}

func (r *runner) advance(ms int64) {
	if ms < 0 {
		ms = 0
	}
	r.timestamp += ms
}

func (r *runner) processingDelay(minMs, maxMs int64) int64 {
	if maxMs <= minMs {
		return minMs
	}
	return minMs + r.rng.Int63n(maxMs-minMs+1)
}

func (r *runner) failoverDelay() int64 {
	return 20 + r.rng.Int63n(181)
}

func (r *runner) retryDelay() int64 {
	return 20 + r.rng.Int63n(31)
}

func evaluateFirewall(node topology.Node, sourceIP, destinationIP, protocol string, port int) (bool, string) {
	rules := anySlice(node.Config["rules"])
	sort.SliceStable(rules, func(i, j int) bool {
		return intValue(anyMap(rules[i])["priority"], 1000) < intValue(anyMap(rules[j])["priority"], 1000)
	})
	for _, item := range rules {
		rule := anyMap(item)
		if !strings.EqualFold(defaultString(rule["protocol"], protocol), protocol) {
			continue
		}
		if intValue(rule["port"], port) != port {
			continue
		}
		if !cidrMatches(defaultString(rule["source"], "0.0.0.0/0"), sourceIP) {
			continue
		}
		if !cidrMatches(defaultString(rule["destination"], "0.0.0.0/0"), destinationIP) {
			continue
		}
		action := defaultString(rule["action"], "deny")
		priority := intValue(rule["priority"], 1000)
		allowed := action == "allow"
		return allowed, fmt.Sprintf("Firewall rule #%d %s tcp/%d from %s to %s", priority, strings.ToUpper(action), port, sourceIP, destinationIP)
	}
	defaultPolicy := defaultString(node.Config["defaultPolicy"], "deny")
	return defaultPolicy == "allow", "Firewall default policy " + strings.ToUpper(defaultPolicy)
}

func targetHost(target string) string {
	parsed, err := url.Parse(target)
	if err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	return strings.TrimSpace(target)
}

func nodeName(node topology.Node) string {
	if strings.TrimSpace(node.Name) != "" {
		return strings.TrimSpace(node.Name)
	}
	return node.ID
}

func nodeIP(node topology.Node) string {
	if ip := stringValue(node.Config["ip"]); ip != "" {
		return ip
	}
	for _, iface := range anySlice(node.Config["interfaces"]) {
		if ip := stringValue(anyMap(iface)["ip"]); ip != "" {
			return ip
		}
	}
	return ""
}

type serverPort struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	Service  string `json:"service,omitempty"`
	Status   string `json:"status,omitempty"`
	Source   string `json:"source,omitempty"`
}

type serverPortCheckResult struct {
	Open          bool
	ImplicitOpen  bool
	RequestedPort int
	Protocol      string
	OpenPorts     []serverPort
	MatchedPort   *serverPort
}

func (r *runner) ensureServerPortOpen(node topology.Node, protocol string, port int, sourceID, targetID string) bool {
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
		r.emit(EventServerPortOpen, SeverityInfo, "server port is open", sourceID, targetID, r.packetID, details)
		r.summary.Decisions = append(r.summary.Decisions, fmt.Sprintf("%s listens on %s/%d", nodeName(node), check.Protocol, check.RequestedPort))
		return true
	}
	r.emit(EventServerPortClosed, SeverityError, "server port is closed", sourceID, targetID, r.packetID, details)
	r.fail(fmt.Sprintf("server does not listen on %s/%d", check.Protocol, check.RequestedPort))
	return false
}

func serverPortCheck(node topology.Node, protocol string, port int) serverPortCheckResult {
	normalizedProtocol := strings.ToLower(strings.TrimSpace(protocol))
	if normalizedProtocol == "" {
		normalizedProtocol = "tcp"
	}
	ports, explicit := serverOpenPorts(node)
	if len(ports) == 0 && !explicit {
		return serverPortCheckResult{Open: true, ImplicitOpen: true, RequestedPort: port, Protocol: normalizedProtocol, OpenPorts: ports}
	}
	for i := range ports {
		item := ports[i]
		if item.Protocol == normalizedProtocol && item.Port == port && portStatusOpen(item.Status) {
			return serverPortCheckResult{Open: true, RequestedPort: port, Protocol: normalizedProtocol, OpenPorts: ports, MatchedPort: &item}
		}
	}
	return serverPortCheckResult{Open: false, RequestedPort: port, Protocol: normalizedProtocol, OpenPorts: ports}
}

func serverOpenPorts(node topology.Node) ([]serverPort, bool) {
	ports := make([]serverPort, 0)
	_, hasOpenPorts := node.Config["openPorts"]
	for _, item := range anySlice(node.Config["openPorts"]) {
		if parsed, ok := parseServerPort(item, "openPorts"); ok {
			ports = append(ports, parsed)
		}
	}
	_, hasPorts := node.Config["ports"]
	for _, item := range anySlice(node.Config["ports"]) {
		if parsed, ok := parseServerPort(item, "ports"); ok {
			ports = append(ports, parsed)
		}
	}
	if len(ports) > 0 {
		return ports, true
	}
	if port := intValue(node.Config["port"], 0); port > 0 {
		ports = append(ports, legacyServerPort(node, port, "port"))
	}
	if port := intValue(node.Config["servicePort"], 0); port > 0 {
		ports = append(ports, legacyServerPort(node, port, "servicePort"))
	}
	return ports, hasOpenPorts || hasPorts || len(ports) > 0
}

func parseServerPort(value any, source string) (serverPort, bool) {
	if port := intValue(value, 0); port > 0 {
		return serverPort{Protocol: "tcp", Port: port, Service: wellKnownService(port), Status: "open", Source: source}, true
	}
	m := anyMap(value)
	port := intValue(m["port"], 0)
	if port <= 0 {
		return serverPort{}, false
	}
	protocol := strings.ToLower(strings.TrimSpace(stringValue(m["protocol"])))
	if protocol == "" {
		protocol = "tcp"
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
		status = "open"
	}
	return serverPort{Protocol: protocol, Port: port, Service: service, Status: status, Source: source}, true
}

func legacyServerPort(node topology.Node, port int, source string) serverPort {
	service := strings.TrimSpace(stringValue(node.Config["serviceName"]))
	if service == "" {
		service = wellKnownService(port)
	}
	return serverPort{Protocol: "tcp", Port: port, Service: service, Status: "open", Source: source}
}

func portStatusOpen(status string) bool {
	normalized := strings.ToLower(strings.TrimSpace(status))
	return normalized == "" || normalized == "open" || normalized == "active" || normalized == "healthy"
}

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

func nodeDown(node topology.Node) bool {
	status := node.Status
	if status == "" {
		status = stringValue(node.Config["status"])
	}
	return status == "down" || status == "isolated"
}

func linkDown(link topology.Link) bool {
	status := link.Status
	if status == "" {
		status = stringValue(link.Config["status"])
	}
	return status == "down"
}

func linkInPath(link topology.Link, path []string) bool {
	for i := 0; i < len(path)-1; i++ {
		if (link.SourceNodeID == path[i] && link.TargetNodeID == path[i+1]) || (link.TargetNodeID == path[i] && link.SourceNodeID == path[i+1]) {
			return true
		}
	}
	return false
}

func cidrMatches(cidr, ipValue string) bool {
	if cidr == "" || cidr == "any" || cidr == "0.0.0.0/0" {
		return true
	}
	ip := net.ParseIP(ipValue)
	if ip == nil {
		return false
	}
	if !strings.Contains(cidr, "/") {
		return cidr == ipValue
	}
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return network.Contains(ip)
}

func anyMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func anySlice(value any) []any {
	if value == nil {
		return nil
	}
	if items, ok := value.([]any); ok {
		return items
	}
	return nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func defaultString(value any, fallback string) string {
	if s := stringValue(value); s != "" {
		return s
	}
	return fallback
}

func intValue(value any, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, err := v.Int64()
		if err == nil {
			return int(i)
		}
	case string:
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return fallback
}

func floatValue(value any, fallback float64) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return f
		}
	}
	return fallback
}

func boolValue(value any, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return fallback
}

func errorCodeForMessage(message string) string {
	switch {
	case message == "sourceNodeId is required":
		return "SOURCE_NODE_REQUIRED"
	case message == "source node does not exist":
		return "SOURCE_NODE_NOT_FOUND"
	case message == "source node must be a client":
		return "SOURCE_NODE_MUST_BE_CLIENT"
	case message == "source client is down":
		return "SOURCE_NODE_DOWN"
	case strings.HasPrefix(message, "ping destination not found"):
		return "DESTINATION_NOT_FOUND"
	case strings.HasPrefix(message, "no route from"):
		return "ROUTE_NOT_FOUND"
	case strings.HasPrefix(message, "DNS NXDOMAIN"):
		return "DNS_RECORD_NOT_FOUND"
	case strings.Contains(message, "Firewall"):
		return "FIREWALL_DENIED"
	case strings.HasPrefix(message, "server does not listen on"):
		return "SERVER_PORT_CLOSED"
	case message == "Load balancer has no healthy backends available.":
		return "NO_HEALTHY_BACKENDS"
	case strings.Contains(message, "topology"):
		return "TOPOLOGY_INVALID"
	default:
		return "SIMULATION_FAILED"
	}
}

func errorUserMessage(code string) string {
	switch code {
	case "SOURCE_NODE_REQUIRED":
		return "Выберите Client, от которого нужно отправить request."
	case "SOURCE_NODE_NOT_FOUND":
		return "Source node не найден."
	case "SOURCE_NODE_MUST_BE_CLIENT":
		return "Source node должен быть Client."
	case "SOURCE_NODE_DOWN":
		return "Выбранный Client недоступен."
	case "DESTINATION_NOT_FOUND":
		return "Назначение не найдено."
	case "ROUTE_NOT_FOUND":
		return "Маршрут не найден."
	case "DNS_RECORD_NOT_FOUND":
		return "DNS record отсутствует."
	case "FIREWALL_DENIED":
		return "Firewall заблокировал packet."
	case "SERVER_PORT_CLOSED":
		return "Server does not listen on the requested port."
	case "NO_HEALTHY_BACKENDS":
		return "Нет доступных healthy backend."
	default:
		return "Simulation завершилась ошибкой."
	}
}

func errorSuggestedFix(code string) string {
	switch code {
	case "SERVER_PORT_CLOSED":
		return "Open the requested server port or change the request/backend service port."
	case "SOURCE_NODE_REQUIRED", "SOURCE_NODE_NOT_FOUND", "SOURCE_NODE_MUST_BE_CLIENT", "SOURCE_NODE_DOWN":
		return "Проверьте выбранный source Client и его status."
	case "DESTINATION_NOT_FOUND":
		return "Проверьте target node, hostname или IP."
	case "ROUTE_NOT_FOUND":
		return "Проверьте links, route table, gateway и status nodes."
	case "DNS_RECORD_NOT_FOUND":
		return "Добавьте корректный DNS A record."
	case "FIREWALL_DENIED":
		return "Добавьте allow rule для нужного protocol/port выше deny rule."
	case "NO_HEALTHY_BACKENDS":
		return "Добавьте healthy reachable Server в Load Balancer backend pool."
	default:
		return "Откройте Timeline и Validation Advisor, чтобы найти проблемный stage."
	}
}

func eventID(simulationID string, sequence int64) string {
	return idgen.DeterministicUUID(fmt.Sprintf("%s_evt_%04d", simulationID, sequence))
}
