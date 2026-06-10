package simulation

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/netquest/netquest/backend/internal/topology"
)

func TestBasicEngineDNSLookupSuccess(t *testing.T) {
	result := runScenario(t, Scenario{Type: "dns_lookup", SourceNodeID: "client-1", Target: "api.netquest.local"}, demoTopology())
	if result.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %s: %#v", result.Status, result.Summary.Errors)
	}
	if result.Summary.ResolvedIP != "10.0.2.10" {
		t.Fatalf("unexpected resolved ip: %s", result.Summary.ResolvedIP)
	}
	if !hasEvent(result, EventDNSResponse) {
		t.Fatalf("expected dns.response event")
	}
}

func TestBasicEngineDNSLookupNXDOMAIN(t *testing.T) {
	result := runScenario(t, Scenario{Type: "dns_lookup", SourceNodeID: "client-1", Target: "missing.netquest.local"}, demoTopology())
	if result.Status != StatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if !hasEvent(result, EventDNSError) || !hasEvent(result, EventSimulationFailed) {
		t.Fatalf("expected dns.error and simulation.failed events: %#v", result.Events)
	}
}

func TestBasicEnginePingSuccess(t *testing.T) {
	result := runScenario(t, Scenario{Type: "icmp_ping", SourceNodeID: "client-1", Target: "server-1"}, demoTopology())
	if result.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %s: %#v", result.Status, result.Summary.Errors)
	}
	if result.Summary.TotalLatencyMs <= 0 {
		t.Fatalf("expected positive rtt")
	}
	if !hasEvent(result, EventRouteSelected) || !hasEvent(result, EventPacketDelivered) {
		t.Fatalf("expected route and delivery events")
	}
}

func TestSimulationUsesProvidedSourceClient(t *testing.T) {
	result := runScenario(t, Scenario{Type: "icmp_ping", SourceNodeID: "client-2", Target: "server-1"}, demoTopologyWithClient2())
	if result.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %s: %#v", result.Status, result.Summary.Errors)
	}
	if result.Summary.SourceNodeID != "client-2" {
		t.Fatalf("expected summary source client-2, got %s", result.Summary.SourceNodeID)
	}
	if !hasEventWithSource(result, EventRouteSelected, "client-2") {
		t.Fatalf("expected route.selected event from client-2")
	}
}

func TestSimulationRejectsMissingSource(t *testing.T) {
	result := runScenario(t, Scenario{Type: "icmp_ping", Target: "server-1"}, demoTopology())
	if result.Status != StatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if len(result.Summary.Errors) == 0 || result.Summary.Errors[0] != "sourceNodeId is required" {
		t.Fatalf("unexpected source error: %#v", result.Summary.Errors)
	}
}

func TestSimulationRejectsNonClientSource(t *testing.T) {
	result := runScenario(t, Scenario{Type: "icmp_ping", SourceNodeID: "server-1", Target: "server-2"}, demoTopology())
	if result.Status != StatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if len(result.Summary.Errors) == 0 || result.Summary.Errors[0] != "source node must be a client" {
		t.Fatalf("unexpected source error: %#v", result.Summary.Errors)
	}
}

func TestSimulationRejectsDownSourceClient(t *testing.T) {
	result := runScenario(t, Scenario{Type: "icmp_ping", SourceNodeID: "client-2", Target: "server-1"}, demoTopologyWithClient2Down())
	if result.Status != StatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if len(result.Summary.Errors) == 0 || result.Summary.Errors[0] != "source client is down" {
		t.Fatalf("unexpected source error: %#v", result.Summary.Errors)
	}
}

func TestLatencyBreakdownReturnedForPing(t *testing.T) {
	result := runScenario(t, Scenario{Type: "icmp_ping", SourceNodeID: "client-1", Target: "server-1"}, demoTopology())
	if !hasLatencyStage(result, "route_lookup") || !hasLatencyStage(result, "icmp_rtt") {
		t.Fatalf("expected ping latency breakdown, got %#v", result.Summary.LatencyBreakdown)
	}
	if result.Summary.LatencyFormula == "" || result.Summary.Seed != 2 {
		t.Fatalf("expected latency formula and seed in summary: %#v", result.Summary)
	}
}

func TestLatencyBreakdownReturnedForHTTPS(t *testing.T) {
	result := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, demoTopology())
	for _, stage := range []string{"dns_lookup", "route_lookup", "firewall_decision", "tcp_handshake", "tls_handshake", "load_balancer_decision", "backend_delivery"} {
		if !hasLatencyStage(result, stage) {
			t.Fatalf("expected latency stage %s, got %#v", stage, result.Summary.LatencyBreakdown)
		}
	}
}

func TestBasicEngineRouteNotFound(t *testing.T) {
	result := runScenario(t, Scenario{Type: "icmp_ping", SourceNodeID: "client-1", Target: "server-1"}, `{
		"nodes": [
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"server-1","type":"server","config":{"ip":"10.0.2.21"}}
		],
		"links": []
	}`)
	if result.Status != StatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if !hasEvent(result, EventRouteNotFound) {
		t.Fatalf("expected route.not_found event")
	}
}

func TestBasicEngineFirewallDeny(t *testing.T) {
	result := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, firewallDenyTopology())
	if result.Status != StatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if !hasEvent(result, EventFirewallDenied) {
		t.Fatalf("expected firewall.denied event")
	}
}

func TestBasicEngineLoadBalancerFailover(t *testing.T) {
	result := runScenario(t, Scenario{Type: "failover_demo", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, demoTopologyWithServer1Down())
	if result.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %s: %#v", result.Status, result.Summary.Errors)
	}
	if result.Summary.SelectedBackend != "server-2" {
		t.Fatalf("expected server-2 failover backend, got %s", result.Summary.SelectedBackend)
	}
	if !hasEvent(result, EventFailoverTriggered) || !hasEvent(result, EventLBBackendUnhealthy) {
		t.Fatalf("expected failover and unhealthy backend events")
	}
}

func TestEventTimestampsAreMonotonic(t *testing.T) {
	result := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, demoTopology())
	var previous int64 = -1
	for _, event := range result.Events {
		if event.TimestampMs < previous {
			t.Fatalf("timestamps must be monotonic: previous=%d current=%d event=%s", previous, event.TimestampMs, event.Type)
		}
		previous = event.TimestampMs
	}
}

func TestTotalLatencyChangesWhenLinkLatencyChanges(t *testing.T) {
	base := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, demoTopology())
	slowerTopology := strings.Replace(demoTopology(), `"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":4}`, `"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":200}`, 1)
	slower := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, slowerTopology)
	if slower.Summary.TotalLatencyMs <= base.Summary.TotalLatencyMs {
		t.Fatalf("expected higher link latency to increase total latency: base=%d slower=%d", base.Summary.TotalLatencyMs, slower.Summary.TotalLatencyMs)
	}
}

func TestTimelineUsesLinkLatency(t *testing.T) {
	base := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, demoTopology())
	slowerTopology := strings.Replace(demoTopology(), `"id":"l3","sourceNodeId":"router-1","targetNodeId":"firewall-1","config":{"latencyMs":8}`, `"id":"l3","sourceNodeId":"router-1","targetNodeId":"firewall-1","config":{"latencyMs":80}`, 1)
	slower := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, slowerTopology)
	if timestampOf(t, slower, EventPacketDelivered) <= timestampOf(t, base, EventPacketDelivered) {
		t.Fatalf("expected packet.delivered timestamp to grow with path latency")
	}
}

func TestPacketLossDeterministicBySeed(t *testing.T) {
	lossy := strings.Replace(demoTopology(), `"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5}`, `"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5,"packetLossPercent":100}`, 1)
	first := runScenario(t, Scenario{Type: "icmp_ping", SourceNodeID: "client-1", Target: "server-1"}, lossy)
	second := runScenario(t, Scenario{Type: "icmp_ping", SourceNodeID: "client-1", Target: "server-1"}, lossy)
	if first.Status != StatusFailed || second.Status != StatusFailed {
		t.Fatalf("expected deterministic packet loss to fail both runs")
	}
	if len(first.Events) != len(second.Events) || first.Summary.Errors[0] != second.Summary.Errors[0] {
		t.Fatalf("expected packet loss result to be reproducible")
	}
}

func TestLBSelectsHealthyBackend(t *testing.T) {
	result := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, demoTopology())
	if result.Summary.SelectedBackendNodeID != "server-1" {
		t.Fatalf("expected server-1, got %s", result.Summary.SelectedBackendNodeID)
	}
}

func TestLBSelectsNewlyAddedBackend(t *testing.T) {
	result := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, demoTopologyWithServer3())
	if result.Summary.SelectedBackendNodeID != "server-3" {
		t.Fatalf("expected server-3, got %s", result.Summary.SelectedBackendNodeID)
	}
}

func TestLBNoHealthyBackends(t *testing.T) {
	result := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, demoTopologyWithAllBackendsDown())
	if result.Status != StatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if !hasEvent(result, EventLBBackendUnhealthy) {
		t.Fatalf("expected lb.backend.unhealthy event")
	}
	if len(result.Summary.Errors) == 0 || result.Summary.Errors[0] != "Load balancer has no healthy backends available." {
		t.Fatalf("unexpected error: %#v", result.Summary.Errors)
	}
}

func TestLBReportsSkippedBackends(t *testing.T) {
	result := runScenario(t, Scenario{Type: "failover_demo", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, demoTopologyWithServer1Down())
	if len(result.Summary.SkippedBackends) == 0 {
		t.Fatalf("expected skipped backends in summary")
	}
	if result.Summary.SkippedBackends[0].NodeID != "server-1" {
		t.Fatalf("expected server-1 skipped, got %#v", result.Summary.SkippedBackends)
	}
}

func TestDirectServerRequiresOpenHTTPSPort(t *testing.T) {
	result := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, directServerTopology(`[{"protocol":"tcp","port":443,"service":"HTTPS","status":"open"}]`))
	if result.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %s: %#v", result.Status, result.Summary.Errors)
	}
	if !hasEvent(result, EventServerPortOpen) {
		t.Fatalf("expected server.port.open event")
	}
	if result.Summary.ProtocolDetails.Server["open"] != true {
		t.Fatalf("expected protocol details to report open port: %#v", result.Summary.ProtocolDetails.Server)
	}
}

func TestDirectServerClosedHTTPSPortFails(t *testing.T) {
	result := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, directServerTopology(`[{"protocol":"tcp","port":80,"service":"HTTP","status":"open"}]`))
	if result.Status != StatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if !hasEvent(result, EventServerPortClosed) {
		t.Fatalf("expected server.port.closed event")
	}
	if len(result.Summary.Errors) == 0 || result.Summary.Errors[0] != "server does not listen on tcp/443" {
		t.Fatalf("unexpected error: %#v", result.Summary.Errors)
	}
	if result.Summary.ProtocolDetails.Server["open"] != false {
		t.Fatalf("expected protocol details to report closed port: %#v", result.Summary.ProtocolDetails.Server)
	}
}

func TestLBSkipsBackendWithClosedHTTPSPort(t *testing.T) {
	topologyJSON := strings.Replace(demoTopologyWithServer3(), `"id":"server-1","type":"server","config":{"ip":"10.0.2.21","port":443}`, `"id":"server-1","type":"server","config":{"ip":"10.0.2.21","openPorts":[{"protocol":"tcp","port":80,"service":"HTTP","status":"open"}]}`, 1)
	topologyJSON = strings.Replace(topologyJSON, `"id":"server-2","type":"server","config":{"ip":"10.0.2.22","port":443}`, `"id":"server-2","type":"server","status":"down","config":{"ip":"10.0.2.22","port":443}`, 1)
	result := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, topologyJSON)
	if result.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %s: %#v", result.Status, result.Summary.Errors)
	}
	if result.Summary.SelectedBackendNodeID != "server-3" {
		t.Fatalf("expected server-3, got %s", result.Summary.SelectedBackendNodeID)
	}
	if !skipReasonContains(result.Summary.SkippedBackends, "server-1", "server port tcp/443 is closed") {
		t.Fatalf("expected server-1 skipped for closed port, got %#v", result.Summary.SkippedBackends)
	}
}

func TestBrokenLinkForcesAlternativeBackend(t *testing.T) {
	broken := strings.Replace(demoTopology(), `"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":4}`, `"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","status":"down","config":{"latencyMs":4}`, 1)
	result := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, broken)
	if result.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %s: %#v", result.Status, result.Summary.Errors)
	}
	if result.Summary.SelectedBackendNodeID != "server-2" {
		t.Fatalf("expected server-2 when server-1 link is down, got %s", result.Summary.SelectedBackendNodeID)
	}
}

func TestLBDoesNotUseDeletedBackend(t *testing.T) {
	engine := NewBasicEngine(topology.NewValidator())
	_, err := engine.Run(context.Background(), RunRequest{
		SimulationID: "11111111-1111-4111-8111-111111111111",
		Seed:         2,
		Scenario:     Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"},
		Topology:     []byte(demoTopologyWithStaleBackend()),
	})
	if err == nil {
		t.Fatal("expected stale backend topology to be rejected")
	}
	if _, ok := err.(TopologyInvalidError); !ok {
		t.Fatalf("expected TopologyInvalidError, got %T: %v", err, err)
	}
}

func TestHTTPSProtocolDetailsIncludeLayers(t *testing.T) {
	result := runScenario(t, Scenario{Type: "https_request", SourceNodeID: "client-1", Target: "https://api.netquest.local/users"}, demoTopology())
	details := result.Summary.ProtocolDetails
	if len(details.Summary) == 0 || len(details.DNS) == 0 || len(details.Routing) == 0 || len(details.Firewall) == 0 ||
		len(details.TCP) == 0 || len(details.TLS) == 0 || len(details.LoadBalancer) == 0 {
		t.Fatalf("expected protocol inspector details for all HTTPS layers: %#v", details)
	}
}

func TestRouteTableUsesLongestPrefixBeforeMetric(t *testing.T) {
	result := runScenario(t, Scenario{Type: "icmp_ping", SourceNodeID: "client-1", Target: "server-1"}, routeTableTopology(routeTableOptions{
		RouteB: `{"destination":"10.0.9.0/24","gateway":"10.0.2.1","interface":"eth1","metric":50}`,
		RouteC: `{"destination":"10.0.0.0/8","gateway":"10.0.3.1","interface":"eth2","metric":1}`,
	}))
	if result.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %s: %#v", result.Status, result.Summary.Errors)
	}
	if !stringIn(result.Summary.Path, "router-b") || stringIn(result.Summary.Path, "router-c") {
		t.Fatalf("expected longest-prefix route through router-b, got %#v", result.Summary.Path)
	}
}

func TestRouteTableUsesLowerMetricForEqualPrefix(t *testing.T) {
	result := runScenario(t, Scenario{Type: "icmp_ping", SourceNodeID: "client-1", Target: "server-1"}, routeTableTopology(routeTableOptions{
		RouteB: `{"destination":"10.0.9.0/24","gateway":"10.0.2.1","interface":"eth1","metric":50}`,
		RouteC: `{"destination":"10.0.9.0/24","gateway":"10.0.3.1","interface":"eth2","metric":5}`,
	}))
	if result.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %s: %#v", result.Status, result.Summary.Errors)
	}
	if !stringIn(result.Summary.Path, "router-c") || stringIn(result.Summary.Path, "router-b") {
		t.Fatalf("expected lower-metric route through router-c, got %#v", result.Summary.Path)
	}
}

func TestRouteTableFallsBackToReachableBackupRoute(t *testing.T) {
	result := runScenario(t, Scenario{Type: "icmp_ping", SourceNodeID: "client-1", Target: "server-1"}, routeTableTopology(routeTableOptions{
		RouteB:     `{"destination":"10.0.9.0/24","gateway":"10.0.2.1","interface":"eth1","metric":5}`,
		RouteC:     `{"destination":"10.0.9.0/24","gateway":"10.0.3.1","interface":"eth2","metric":50}`,
		LinkBDown:  true,
		LinkCLatMs: 12,
	}))
	if result.Status != StatusCompleted {
		t.Fatalf("expected completed status through backup route, got %s: %#v", result.Status, result.Summary.Errors)
	}
	if !stringIn(result.Summary.Path, "router-c") || stringIn(result.Summary.Path, "router-b") {
		t.Fatalf("expected backup route through router-c after primary link down, got %#v", result.Summary.Path)
	}
}

func TestRouteTableMissingDefaultGatewayIsHelpful(t *testing.T) {
	result := runScenario(t, Scenario{Type: "icmp_ping", SourceNodeID: "client-1", Target: "server-1"}, strings.Replace(routeTableTopology(routeTableOptions{
		RouteB: `{"destination":"10.0.9.0/24","gateway":"10.0.2.1","interface":"eth1","metric":10}`,
		RouteC: `{"destination":"10.0.9.0/24","gateway":"10.0.3.1","interface":"eth2","metric":20}`,
	}), `"cidr":"10.0.1.10/24","defaultGateway":"10.0.1.1"`, `"cidr":"10.0.1.10/24"`, 1))
	if result.Status != StatusFailed {
		t.Fatalf("expected route failure, got %s", result.Status)
	}
	if !hasEvent(result, EventRouteNotFound) {
		t.Fatalf("expected route.not_found event")
	}
	if result.Summary.ProtocolDetails.Routing["explanation"] == "" {
		t.Fatalf("expected routing explanation in protocol details: %#v", result.Summary.ProtocolDetails.Routing)
	}
}

func runScenario(t *testing.T, scenario Scenario, topologyJSON string) RunResult {
	t.Helper()
	engine := NewBasicEngine(topology.NewValidator())
	result, err := engine.Run(context.Background(), RunRequest{
		SimulationID: "11111111-1111-4111-8111-111111111111",
		Seed:         2,
		Scenario:     scenario,
		Topology:     []byte(topologyJSON),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	return result
}

func hasEvent(result RunResult, eventType EventType) bool {
	for _, event := range result.Events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func hasEventWithSource(result RunResult, eventType EventType, sourceNodeID string) bool {
	for _, event := range result.Events {
		if event.Type == eventType && event.SourceNodeID == sourceNodeID {
			return true
		}
	}
	return false
}

func hasLatencyStage(result RunResult, stage string) bool {
	for _, item := range result.Summary.LatencyBreakdown {
		if item.Stage == stage && item.DurationMs >= 0 {
			return true
		}
	}
	return false
}

func timestampOf(t *testing.T, result RunResult, eventType EventType) int64 {
	t.Helper()
	for _, event := range result.Events {
		if event.Type == eventType {
			return event.TimestampMs
		}
	}
	t.Fatalf("missing event %s", eventType)
	return 0
}

func stringIn(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func skipReasonContains(items []BackendSkip, nodeID, reason string) bool {
	for _, item := range items {
		if item.NodeID == nodeID && strings.Contains(item.Reason, reason) {
			return true
		}
	}
	return false
}

type routeTableOptions struct {
	RouteB     string
	RouteC     string
	LinkBDown  bool
	LinkCLatMs int
}

func routeTableTopology(options routeTableOptions) string {
	linkBStatus := ""
	if options.LinkBDown {
		linkBStatus = `"status":"down",`
	}
	linkCLatency := options.LinkCLatMs
	if linkCLatency == 0 {
		linkCLatency = 8
	}
	return fmt.Sprintf(`{
		"nodes": [
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10","cidr":"10.0.1.10/24","defaultGateway":"10.0.1.1"}},
			{"id":"router-a","type":"router","config":{"ip":"10.0.1.1","cidr":"10.0.1.1/24","routes":[%s,%s]}},
			{"id":"router-b","type":"router","config":{"ip":"10.0.2.1","cidr":"10.0.9.1/24"}},
			{"id":"router-c","type":"router","config":{"ip":"10.0.3.1","cidr":"10.0.9.1/24"}},
			{"id":"server-1","type":"server","config":{"ip":"10.0.9.20","cidr":"10.0.9.20/24"}}
		],
		"links": [
			{"id":"l-client-a","sourceNodeId":"client-1","targetNodeId":"router-a","config":{"latencyMs":2}},
			{"id":"l-a-b","sourceNodeId":"router-a","targetNodeId":"router-b",%s"config":{"latencyMs":3}},
			{"id":"l-a-c","sourceNodeId":"router-a","targetNodeId":"router-c","config":{"latencyMs":%d}},
			{"id":"l-b-server","sourceNodeId":"router-b","targetNodeId":"server-1","config":{"latencyMs":3}},
			{"id":"l-c-server","sourceNodeId":"router-c","targetNodeId":"server-1","config":{"latencyMs":4}}
		]
	}`, options.RouteB, options.RouteC, linkBStatus, linkCLatency)
}

func directServerTopology(openPorts string) string {
	return fmt.Sprintf(`{
		"nodes": [
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"dns-1","type":"dns","config":{"ip":"10.0.1.53","records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.21","ttl":300}]}},
			{"id":"router-1","type":"router","config":{"ip":"10.0.1.1"}},
			{"id":"firewall-1","type":"firewall","config":{"ip":"10.0.1.254","defaultPolicy":"deny","rules":[{"priority":100,"action":"allow","protocol":"tcp","source":"10.0.1.0/24","destination":"10.0.2.21/32","port":443}]}},
			{"id":"server-1","type":"server","config":{"ip":"10.0.2.21","openPorts":%s}}
		],
		"links": [
			{"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5}},
			{"id":"l2","sourceNodeId":"client-1","targetNodeId":"dns-1","config":{"latencyMs":2}},
			{"id":"l3","sourceNodeId":"router-1","targetNodeId":"firewall-1","config":{"latencyMs":8}},
			{"id":"l4","sourceNodeId":"firewall-1","targetNodeId":"server-1","config":{"latencyMs":12}}
		]
	}`, openPorts)
}

func demoTopology() string {
	return `{
		"nodes": [
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"dns-1","type":"dns","config":{"ip":"10.0.1.53","records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.10","ttl":300}]}},
			{"id":"router-1","type":"router","config":{"ip":"10.0.1.1"}},
			{"id":"firewall-1","type":"firewall","config":{"ip":"10.0.1.254","defaultPolicy":"deny","rules":[{"priority":100,"action":"allow","protocol":"tcp","source":"10.0.1.0/24","destination":"10.0.2.10/32","port":443}]}},
			{"id":"lb-1","type":"load_balancer","config":{"ip":"10.0.2.10","algorithm":"round_robin","backends":[{"nodeId":"server-1","healthy":true},{"nodeId":"server-2","healthy":true}]}},
			{"id":"server-1","type":"server","config":{"ip":"10.0.2.21","status":"healthy"}},
			{"id":"server-2","type":"server","config":{"ip":"10.0.2.22","status":"healthy"}}
		],
		"links": [
			{"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5}},
			{"id":"l2","sourceNodeId":"client-1","targetNodeId":"dns-1","config":{"latencyMs":2}},
			{"id":"l3","sourceNodeId":"router-1","targetNodeId":"firewall-1","config":{"latencyMs":8}},
			{"id":"l4","sourceNodeId":"firewall-1","targetNodeId":"lb-1","config":{"latencyMs":12}},
			{"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":4}},
			{"id":"l6","sourceNodeId":"lb-1","targetNodeId":"server-2","config":{"latencyMs":6}}
		]
	}`
}

func demoTopologyWithClient2() string {
	withNode := strings.Replace(demoTopology(), `{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},`, `{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"client-2","type":"client","config":{"ip":"10.0.1.11"}},`, 1)
	return strings.Replace(withNode, `{"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5}},`, `{"id":"l0","sourceNodeId":"client-2","targetNodeId":"router-1","config":{"latencyMs":7}},
			{"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5}},`, 1)
}

func demoTopologyWithClient2Down() string {
	return strings.Replace(demoTopologyWithClient2(), `{"id":"client-2","type":"client","config":{"ip":"10.0.1.11"}},`, `{"id":"client-2","type":"client","status":"down","config":{"ip":"10.0.1.11"}},`, 1)
}

func firewallDenyTopology() string {
	return `{
		"nodes": [
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"dns-1","type":"dns","config":{"records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.10","ttl":300}]}},
			{"id":"router-1","type":"router","config":{}},
			{"id":"firewall-1","type":"firewall","config":{"defaultPolicy":"deny","rules":[{"priority":100,"action":"deny","protocol":"tcp","source":"10.0.1.0/24","destination":"10.0.2.10/32","port":443}]}},
			{"id":"lb-1","type":"load_balancer","config":{"ip":"10.0.2.10","backends":[{"nodeId":"server-1","healthy":true}]}},
			{"id":"server-1","type":"server","config":{"ip":"10.0.2.21"}}
		],
		"links": [
			{"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5}},
			{"id":"l2","sourceNodeId":"client-1","targetNodeId":"dns-1","config":{"latencyMs":2}},
			{"id":"l3","sourceNodeId":"router-1","targetNodeId":"firewall-1","config":{"latencyMs":8}},
			{"id":"l4","sourceNodeId":"firewall-1","targetNodeId":"lb-1","config":{"latencyMs":12}},
			{"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":4}}
		]
	}`
}

func demoTopologyWithServer1Down() string {
	return `{
		"nodes": [
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"dns-1","type":"dns","config":{"records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.10","ttl":300}]}},
			{"id":"router-1","type":"router","config":{}},
			{"id":"firewall-1","type":"firewall","config":{"defaultPolicy":"deny","rules":[{"priority":100,"action":"allow","protocol":"tcp","source":"10.0.1.0/24","destination":"10.0.2.10/32","port":443}]}},
			{"id":"lb-1","type":"load_balancer","config":{"ip":"10.0.2.10","algorithm":"round_robin","backends":[{"nodeId":"server-1","healthy":true},{"nodeId":"server-2","healthy":true}]}},
			{"id":"server-1","type":"server","status":"down","config":{"ip":"10.0.2.21"}},
			{"id":"server-2","type":"server","config":{"ip":"10.0.2.22","status":"healthy"}}
		],
		"links": [
			{"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5}},
			{"id":"l2","sourceNodeId":"client-1","targetNodeId":"dns-1","config":{"latencyMs":2}},
			{"id":"l3","sourceNodeId":"router-1","targetNodeId":"firewall-1","config":{"latencyMs":8}},
			{"id":"l4","sourceNodeId":"firewall-1","targetNodeId":"lb-1","config":{"latencyMs":12}},
			{"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":4}},
			{"id":"l6","sourceNodeId":"lb-1","targetNodeId":"server-2","config":{"latencyMs":6}}
		]
	}`
}

func demoTopologyWithServer3() string {
	return `{
		"nodes": [
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"dns-1","type":"dns","config":{"ip":"10.0.1.53","records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.10","ttl":300}]}},
			{"id":"router-1","type":"router","config":{"ip":"10.0.1.1"}},
			{"id":"firewall-1","type":"firewall","config":{"ip":"10.0.1.254","defaultPolicy":"deny","rules":[{"priority":100,"action":"allow","protocol":"tcp","source":"10.0.1.0/24","destination":"10.0.2.10/32","port":443}]}},
			{"id":"lb-1","type":"load_balancer","config":{"ip":"10.0.2.10","algorithm":"round_robin","backends":[{"nodeId":"server-1","enabled":true},{"nodeId":"server-2","enabled":true},{"nodeId":"server-3","enabled":true}]}},
			{"id":"server-1","type":"server","config":{"ip":"10.0.2.21","port":443}},
			{"id":"server-2","type":"server","config":{"ip":"10.0.2.22","port":443}},
			{"id":"server-3","type":"server","config":{"ip":"10.0.2.23","port":443}}
		],
		"links": [
			{"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5}},
			{"id":"l2","sourceNodeId":"client-1","targetNodeId":"dns-1","config":{"latencyMs":2}},
			{"id":"l3","sourceNodeId":"router-1","targetNodeId":"firewall-1","config":{"latencyMs":8}},
			{"id":"l4","sourceNodeId":"firewall-1","targetNodeId":"lb-1","config":{"latencyMs":12}},
			{"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":4}},
			{"id":"l6","sourceNodeId":"lb-1","targetNodeId":"server-2","config":{"latencyMs":6}},
			{"id":"l7","sourceNodeId":"lb-1","targetNodeId":"server-3","config":{"latencyMs":9}}
		]
	}`
}

func demoTopologyWithAllBackendsDown() string {
	return `{
		"nodes": [
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"dns-1","type":"dns","config":{"records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.10","ttl":300}]}},
			{"id":"router-1","type":"router","config":{}},
			{"id":"firewall-1","type":"firewall","config":{"defaultPolicy":"allow"}},
			{"id":"lb-1","type":"load_balancer","config":{"ip":"10.0.2.10","backends":[{"nodeId":"server-1","enabled":true},{"nodeId":"server-2","enabled":true}]}},
			{"id":"server-1","type":"server","status":"down","config":{"ip":"10.0.2.21"}},
			{"id":"server-2","type":"server","status":"down","config":{"ip":"10.0.2.22"}}
		],
		"links": [
			{"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5}},
			{"id":"l2","sourceNodeId":"client-1","targetNodeId":"dns-1","config":{"latencyMs":2}},
			{"id":"l3","sourceNodeId":"router-1","targetNodeId":"firewall-1","config":{"latencyMs":8}},
			{"id":"l4","sourceNodeId":"firewall-1","targetNodeId":"lb-1","config":{"latencyMs":12}},
			{"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":4}},
			{"id":"l6","sourceNodeId":"lb-1","targetNodeId":"server-2","config":{"latencyMs":6}}
		]
	}`
}

func demoTopologyWithStaleBackend() string {
	return `{
		"nodes": [
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"dns-1","type":"dns","config":{"records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.10","ttl":300}]}},
			{"id":"router-1","type":"router","config":{}},
			{"id":"firewall-1","type":"firewall","config":{"defaultPolicy":"allow"}},
			{"id":"lb-1","type":"load_balancer","config":{"ip":"10.0.2.10","backends":[{"nodeId":"server-deleted","enabled":true}]}},
			{"id":"server-1","type":"server","config":{"ip":"10.0.2.21"}}
		],
		"links": [
			{"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5}},
			{"id":"l2","sourceNodeId":"client-1","targetNodeId":"dns-1","config":{"latencyMs":2}},
			{"id":"l3","sourceNodeId":"router-1","targetNodeId":"firewall-1","config":{"latencyMs":8}},
			{"id":"l4","sourceNodeId":"firewall-1","targetNodeId":"lb-1","config":{"latencyMs":12}},
			{"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":4}}
		]
	}`
}
