package tests

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
)

// TestDNSLookup: имя разрешается в A-запись резолвера; неизвестное имя — NXDOMAIN
// и провал симуляции с событиями dns.error и simulation.failed.
func TestDNSLookup(t *testing.T) {
	t.Parallel()

	ok := runScenario(t, model.Scenario{Type: model.ScenarioDNSLookup, SourceNodeID: "client-1", Target: "api.netquest.local"}, demoTopology())
	require.Equal(t, model.SimulationStatusCompleted, ok.Status, ok.Summary.Errors)
	assert.Equal(t, "10.0.2.10", ok.Summary.ResolvedIP)
	assert.True(t, hasEvent(ok, model.EventDNSResponse))

	nx := runScenario(t, model.Scenario{Type: model.ScenarioDNSLookup, SourceNodeID: "client-1", Target: "missing.netquest.local"}, demoTopology())
	require.Equal(t, model.SimulationStatusFailed, nx.Status)
	assert.True(t, hasEvent(nx, model.EventDNSError))
	assert.True(t, hasEvent(nx, model.EventSimulationFailed))
}

// TestPing: успешный ping даёт положительный RTT, события маршрута и доставки
// и этапы задержки route_lookup и icmp_rtt.
func TestPing(t *testing.T) {
	t.Parallel()

	result := runScenario(t, ping("server-1"), demoTopology())

	require.Equal(t, model.SimulationStatusCompleted, result.Status, result.Summary.Errors)
	assert.Positive(t, result.Summary.TotalLatencyMs)
	assert.True(t, hasEvent(result, model.EventRouteSelected))
	assert.True(t, hasEvent(result, model.EventPacketDelivered))
	assert.True(t, hasLatencyStage(result, "route_lookup"))
	assert.True(t, hasLatencyStage(result, "icmp_rtt"))
	assert.NotEmpty(t, result.Summary.LatencyFormula)
	assert.Equal(t, int64(testSeed), result.Summary.Seed)
}

// TestSourceValidation: источник обязан быть существующим включённым client;
// иначе симуляция проваливается первой же ошибкой с понятным текстом.
func TestSourceValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		scenario  model.Scenario
		topology  string
		wantError string
	}{
		{"нет источника", model.Scenario{Type: model.ScenarioICMPPing, Target: "server-1"}, demoTopology(), "sourceNodeId is required"},
		{"источник не client", model.Scenario{Type: model.ScenarioICMPPing, SourceNodeID: "server-1", Target: "server-2"}, demoTopology(), "source node must be a client"},
		{"источник выключен", model.Scenario{Type: model.ScenarioICMPPing, SourceNodeID: "client-2", Target: "server-1"}, demoTopologyWithClient2Down(), "source client is down"},
		{"источника нет в топологии", model.Scenario{Type: model.ScenarioICMPPing, SourceNodeID: "ghost", Target: "server-1"}, demoTopology(), "source node does not exist"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := runScenario(t, tc.scenario, tc.topology)

			require.Equal(t, model.SimulationStatusFailed, result.Status)
			require.NotEmpty(t, result.Summary.Errors)
			assert.Equal(t, tc.wantError, result.Summary.Errors[0])
		})
	}
}

// TestUsesProvidedSourceClient: пакет идёт от выбранного клиента, а не от первого в топологии.
func TestUsesProvidedSourceClient(t *testing.T) {
	t.Parallel()

	result := runScenario(t, model.Scenario{Type: model.ScenarioICMPPing, SourceNodeID: "client-2", Target: "server-1"}, demoTopologyWithClient2())

	require.Equal(t, model.SimulationStatusCompleted, result.Status, result.Summary.Errors)
	assert.Equal(t, "client-2", result.Summary.SourceNodeID)
	assert.True(t, hasEventWithSource(result, model.EventRouteSelected, "client-2"))
}

// TestHTTPSLatencyBreakdown: HTTPS-запрос через Load Balancer раскладывает задержку
// по всем этапам, а протокольный разбор заполнен по всем слоям.
func TestHTTPSLatencyBreakdown(t *testing.T) {
	t.Parallel()

	result := runScenario(t, https(), demoTopology())

	for _, stage := range []string{"dns_lookup", "route_lookup", "firewall_decision", "tcp_handshake", "tls_handshake", "load_balancer_decision", "backend_delivery"} {
		assert.True(t, hasLatencyStage(result, stage), "нет этапа %s", stage)
	}

	details := result.Summary.ProtocolDetails
	assert.NotEmpty(t, details.Summary)
	assert.NotEmpty(t, details.DNS)
	assert.NotEmpty(t, details.Routing)
	assert.NotEmpty(t, details.Firewall)
	assert.NotEmpty(t, details.TCP)
	assert.NotEmpty(t, details.TLS)
	assert.NotEmpty(t, details.LoadBalancer)
}

// TestEventTimestampsAreMonotonic: виртуальное время событий не идёт назад.
func TestEventTimestampsAreMonotonic(t *testing.T) {
	t.Parallel()

	result := runScenario(t, https(), demoTopology())

	previous := int64(-1)
	for _, event := range result.Events {
		assert.GreaterOrEqual(t, event.TimestampMs, previous, "событие %s", event.Type)
		previous = event.TimestampMs
	}
}

// TestLinkLatencyAffectsTiming: задержка канала влияет и на итоговую задержку,
// и на время доставки в Timeline.
func TestLinkLatencyAffectsTiming(t *testing.T) {
	t.Parallel()

	base := runScenario(t, https(), demoTopology())

	slowBackend := strings.Replace(demoTopology(), `"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":4}`, `"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":200}`, 1)
	slower := runScenario(t, https(), slowBackend)
	assert.Greater(t, slower.Summary.TotalLatencyMs, base.Summary.TotalLatencyMs)

	slowFirewall := strings.Replace(demoTopology(), `"id":"l3","sourceNodeId":"router-1","targetNodeId":"firewall-1","config":{"latencyMs":8}`, `"id":"l3","sourceNodeId":"router-1","targetNodeId":"firewall-1","config":{"latencyMs":80}`, 1)
	slowerPath := runScenario(t, https(), slowFirewall)
	assert.Greater(t, timestampOf(t, slowerPath, model.EventPacketDelivered), timestampOf(t, base, model.EventPacketDelivered))
}

// TestPacketLossIsDeterministic: потеря пакета решается генератором с seed —
// два прогона одной топологии дают одинаковый результат.
func TestPacketLossIsDeterministic(t *testing.T) {
	t.Parallel()

	lossy := strings.Replace(demoTopology(), `"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5}`, `"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5,"packetLossPercent":100}`, 1)

	first := runScenario(t, ping("server-1"), lossy)
	second := runScenario(t, ping("server-1"), lossy)

	require.Equal(t, model.SimulationStatusFailed, first.Status)
	require.Equal(t, model.SimulationStatusFailed, second.Status)
	assert.Len(t, second.Events, len(first.Events))
	assert.Equal(t, first.Summary.Errors[0], second.Summary.Errors[0])
}

// TestRouteNotFound: сеть не доставила пакет — это не ошибка движка,
// а результат failed с объясняющим событием route.not_found.
func TestRouteNotFound(t *testing.T) {
	t.Parallel()

	result := runScenario(t, ping("server-1"), `{
		"nodes": [
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"server-1","type":"server","config":{"ip":"10.0.2.21"}}
		],
		"links": []
	}`)

	require.Equal(t, model.SimulationStatusFailed, result.Status)
	assert.True(t, hasEvent(result, model.EventRouteNotFound))
}

// TestFirewallDeny: правило deny на tcp/443 отбрасывает пакет — firewall.denied и failed.
func TestFirewallDeny(t *testing.T) {
	t.Parallel()

	result := runScenario(t, https(), firewallDenyTopology())

	require.Equal(t, model.SimulationStatusFailed, result.Status)
	assert.True(t, hasEvent(result, model.EventFirewallDenied))
}

// TestLoadBalancerSelection: выбор сервера Load Balancer'ом — исправный,
// доступный и слушающий tcp/443; выключенные и недоступные пропускаются с причиной.
func TestLoadBalancerSelection(t *testing.T) {
	t.Parallel()

	closedPort := strings.Replace(demoTopologyWithServer3(), `"id":"server-1","type":"server","config":{"ip":"10.0.2.21","port":443}`, `"id":"server-1","type":"server","config":{"ip":"10.0.2.21","openPorts":[{"protocol":"tcp","port":80,"service":"HTTP","status":"open"}]}`, 1)
	closedPort = strings.Replace(closedPort, `"id":"server-2","type":"server","config":{"ip":"10.0.2.22","port":443}`, `"id":"server-2","type":"server","status":"down","config":{"ip":"10.0.2.22","port":443}`, 1)
	brokenLink := strings.Replace(demoTopology(), `"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":4}`, `"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","status":"down","config":{"latencyMs":4}`, 1)

	tests := []struct {
		name         string
		scenario     model.Scenario
		topology     string
		wantSelected string
		wantSkipped  string // ID сервера, который должен быть пропущен; пусто — не проверяется
		wantReason   string
	}{
		{name: "исправный сервер", scenario: https(), topology: demoTopology(), wantSelected: "server-1"},
		{name: "новый сервер в пуле", scenario: https(), topology: demoTopologyWithServer3(), wantSelected: "server-3"},
		{
			name: "failover мимо выключенного", topology: demoTopologyWithServer1Down(), wantSelected: "server-2",
			scenario:    model.Scenario{Type: model.ScenarioFailoverDemo, SourceNodeID: "client-1", Target: apiURL},
			wantSkipped: "server-1", wantReason: "node is down",
		},
		{
			name: "закрытый порт 443", scenario: https(), topology: closedPort, wantSelected: "server-3",
			wantSkipped: "server-1", wantReason: "server port tcp/443 is closed",
		},
		{
			name: "канал до сервера выключен", scenario: https(), topology: brokenLink, wantSelected: "server-2",
			wantSkipped: "server-1", wantReason: "no active path from load balancer",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := runScenario(t, tc.scenario, tc.topology)

			require.Equal(t, model.SimulationStatusCompleted, result.Status, result.Summary.Errors)
			assert.Equal(t, tc.wantSelected, result.Summary.SelectedBackendNodeID)

			if tc.wantSkipped != "" {
				assert.True(t, skipReasonContains(result.Summary.SkippedBackends, tc.wantSkipped, tc.wantReason),
					"пропущенные: %#v", result.Summary.SkippedBackends)
				assert.True(t, hasEvent(result, model.EventLBBackendUnhealthy))
			}
		})
	}
}

// TestFailoverDemo: сценарий failover_demo добавляет шаг failover.
func TestFailoverDemo(t *testing.T) {
	t.Parallel()

	result := runScenario(t, model.Scenario{Type: model.ScenarioFailoverDemo, SourceNodeID: "client-1", Target: apiURL}, demoTopologyWithServer1Down())

	require.Equal(t, model.SimulationStatusCompleted, result.Status, result.Summary.Errors)
	assert.Equal(t, "server-2", result.Summary.SelectedBackend)
	assert.True(t, result.Summary.Failover)
	assert.True(t, hasEvent(result, model.EventFailoverTriggered))
}

// TestNoHealthyBackends: все серверы пула выключены — провал с понятной ошибкой.
func TestNoHealthyBackends(t *testing.T) {
	t.Parallel()

	result := runScenario(t, https(), demoTopologyWithAllBackendsDown())

	require.Equal(t, model.SimulationStatusFailed, result.Status)
	assert.True(t, hasEvent(result, model.EventLBBackendUnhealthy))
	require.NotEmpty(t, result.Summary.Errors)
	assert.Equal(t, "Load balancer has no healthy backends available.", result.Summary.Errors[0])
}

// TestDirectServerPort: HTTPS прямо на Server требует открытого tcp/443.
func TestDirectServerPort(t *testing.T) {
	t.Parallel()

	open := runScenario(t, https(), directServerTopology(`[{"protocol":"tcp","port":443,"service":"HTTPS","status":"open"}]`))
	require.Equal(t, model.SimulationStatusCompleted, open.Status, open.Summary.Errors)
	assert.True(t, hasEvent(open, model.EventServerPortOpen))
	assert.Equal(t, true, open.Summary.ProtocolDetails.Server["open"])

	closed := runScenario(t, https(), directServerTopology(`[{"protocol":"tcp","port":80,"service":"HTTP","status":"open"}]`))
	require.Equal(t, model.SimulationStatusFailed, closed.Status)
	assert.True(t, hasEvent(closed, model.EventServerPortClosed))
	assert.Equal(t, "server does not listen on tcp/443", closed.Summary.Errors[0])
	assert.Equal(t, false, closed.Summary.ProtocolDetails.Server["open"])
}

// TestStaleBackendRejected: топологию со ссылкой на удалённый сервер движок
// не считает — это ошибка валидации, а не провал сети.
func TestStaleBackendRejected(t *testing.T) {
	t.Parallel()

	_, err := runEngine(https(), demoTopologyWithStaleBackend())

	require.Error(t, err)

	var invalid errs.TopologyInvalidError
	require.True(t, errors.As(err, &invalid), "ошибка %T: %v", err, err)
	assert.False(t, invalid.Validation.Valid)
}

// TestRouteTables: маршрутизация по таблицам — самый длинный префикс важнее
// метрики, при равном префиксе побеждает меньшая метрика, при недоступном
// следующем хопе — резервный маршрут.
func TestRouteTables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options routeTableOptions
		want    string // роутер, через который должен пройти путь
		notWant string
	}{
		{
			name: "длинный префикс важнее метрики",
			options: routeTableOptions{
				RouteB: `{"destination":"10.0.9.0/24","gateway":"10.0.2.1","interface":"eth1","metric":50}`,
				RouteC: `{"destination":"10.0.0.0/8","gateway":"10.0.3.1","interface":"eth2","metric":1}`,
			},
			want: "router-b", notWant: "router-c",
		},
		{
			name: "при равном префиксе — меньшая метрика",
			options: routeTableOptions{
				RouteB: `{"destination":"10.0.9.0/24","gateway":"10.0.2.1","interface":"eth1","metric":50}`,
				RouteC: `{"destination":"10.0.9.0/24","gateway":"10.0.3.1","interface":"eth2","metric":5}`,
			},
			want: "router-c", notWant: "router-b",
		},
		{
			name: "резервный маршрут при выключенном канале",
			options: routeTableOptions{
				RouteB:     `{"destination":"10.0.9.0/24","gateway":"10.0.2.1","interface":"eth1","metric":5}`,
				RouteC:     `{"destination":"10.0.9.0/24","gateway":"10.0.3.1","interface":"eth2","metric":50}`,
				LinkBDown:  true,
				LinkCLatMs: 12,
			},
			want: "router-c", notWant: "router-b",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := runScenario(t, ping("server-1"), routeTableTopology(tc.options))

			require.Equal(t, model.SimulationStatusCompleted, result.Status, result.Summary.Errors)
			assert.Contains(t, result.Summary.Path, tc.want)
			assert.NotContains(t, result.Summary.Path, tc.notWant)
		})
	}
}

// TestRouteTableMissingDefaultGateway: без шлюза клиент не выходит из подсети,
// и протокольный разбор объясняет почему.
func TestRouteTableMissingDefaultGateway(t *testing.T) {
	t.Parallel()

	topology := strings.Replace(routeTableTopology(routeTableOptions{
		RouteB: `{"destination":"10.0.9.0/24","gateway":"10.0.2.1","interface":"eth1","metric":10}`,
		RouteC: `{"destination":"10.0.9.0/24","gateway":"10.0.3.1","interface":"eth2","metric":20}`,
	}), `"cidr":"10.0.1.10/24","defaultGateway":"10.0.1.1"`, `"cidr":"10.0.1.10/24"`, 1)

	result := runScenario(t, ping("server-1"), topology)

	require.Equal(t, model.SimulationStatusFailed, result.Status)
	assert.True(t, hasEvent(result, model.EventRouteNotFound))
	assert.NotEmpty(t, result.Summary.ProtocolDetails.Routing["explanation"])
}

// TestUnsupportedScenario: неизвестный тип сценария — провал с кодом SIMULATION_FAILED.
func TestUnsupportedScenario(t *testing.T) {
	t.Parallel()

	result := runScenario(t, model.Scenario{Type: "teleport", SourceNodeID: "client-1", Target: "server-1"}, demoTopology())

	require.Equal(t, model.SimulationStatusFailed, result.Status)
	assert.Equal(t, "unsupported scenario type: teleport", result.Summary.Errors[0])
	assert.Equal(t, "SIMULATION_FAILED", result.Summary.Metadata["errorCode"])
}

// TestEventIDsAreDeterministic: ID событий строятся из ID симуляции и номера —
// повторный расчёт даёт те же ID.
func TestEventIDsAreDeterministic(t *testing.T) {
	t.Parallel()

	first := runScenario(t, https(), demoTopology())
	second := runScenario(t, https(), demoTopology())

	require.Len(t, second.Events, len(first.Events))
	for i := range first.Events {
		assert.Equal(t, first.Events[i].ID, second.Events[i].ID)
		assert.Equal(t, int64(i+1), first.Events[i].SequenceNumber)
	}
}

// TestEmptySimulationID: без ID симуляции движок не запускается.
func TestEmptySimulationID(t *testing.T) {
	t.Parallel()

	_, err := runEngineWithID("", ping("server-1"), demoTopology())

	require.Error(t, err)
}
