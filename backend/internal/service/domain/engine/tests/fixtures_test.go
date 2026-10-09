package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/domain/engine"
	"github.com/netquest/netquest/backend/internal/service/domain/validator"
)

// Тесты движка симуляции. Движок настоящий, валидатор настоящий: оба —
// чистые функции над документом, моки здесь только спрятали бы логику.
// Seed фиксирован (testSeed), поэтому каждый прогон даёт те же события.

const (
	testSimulationID = "11111111-1111-4111-8111-111111111111"
	testSeed         = 2
	apiURL           = "https://api.netquest.local/users"
)

// runScenario прогоняет сценарий и требует, чтобы движок не отказался считать.
func runScenario(t *testing.T, scenario model.Scenario, topologyJSON string) model.RunResult {
	t.Helper()

	result, err := runEngine(scenario, topologyJSON)
	require.NoError(t, err)

	return result
}

// runEngine прогоняет сценарий и возвращает и результат, и ошибку движка.
func runEngine(scenario model.Scenario, topologyJSON string) (model.RunResult, error) {
	return runEngineWithID(testSimulationID, scenario, topologyJSON)
}

// runEngineWithID — runEngine с заданным ID симуляции.
func runEngineWithID(simulationID string, scenario model.Scenario, topologyJSON string) (model.RunResult, error) {
	return engine.New(validator.New()).Run(context.Background(), model.RunRequest{
		SimulationID: simulationID,
		Seed:         testSeed,
		Scenario:     scenario,
		Topology:     []byte(topologyJSON),
	})
}

// https — сценарий HTTPS-запроса к API с client-1.
func https() model.Scenario {
	return model.Scenario{Type: model.ScenarioHTTPSRequest, SourceNodeID: "client-1", Target: apiURL}
}

// ping — сценарий ping с client-1 до target.
func ping(target string) model.Scenario {
	return model.Scenario{Type: model.ScenarioICMPPing, SourceNodeID: "client-1", Target: target}
}

func hasEvent(result model.RunResult, eventType model.EventType) bool {
	for _, event := range result.Events {
		if event.Type == eventType {
			return true
		}
	}

	return false
}

func hasEventWithSource(result model.RunResult, eventType model.EventType, sourceNodeID string) bool {
	for _, event := range result.Events {
		if event.Type == eventType && event.SourceNodeID == sourceNodeID {
			return true
		}
	}

	return false
}

func hasLatencyStage(result model.RunResult, stage string) bool {
	for _, item := range result.Summary.LatencyBreakdown {
		if item.Stage == stage && item.DurationMs >= 0 {
			return true
		}
	}

	return false
}

func timestampOf(t *testing.T, result model.RunResult, eventType model.EventType) int64 {
	t.Helper()

	for _, event := range result.Events {
		if event.Type == eventType {
			return event.TimestampMs
		}
	}

	t.Fatalf("нет события %s", eventType)

	return 0
}

func skipReasonContains(items []model.BackendSkip, nodeID, reason string) bool {
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
