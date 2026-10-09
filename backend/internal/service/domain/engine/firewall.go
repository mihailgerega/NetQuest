package engine

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// Значения по умолчанию для правил firewall.
const (
	defaultRulePriority = 1000 // правило без priority проверяется последним
	anyAddress          = "0.0.0.0/0"
	actionAllow         = "allow"
	actionDeny          = "deny" // и действие правила, и политика по умолчанию
)

// firewallAllows пропускает пакет через firewall на пути.
//
// Решение принимает только первый firewall по пути: дальше пакет либо
// отброшен, либо пропущен — второй firewall не проверяется. Решение — событие
// firewall.decision (или firewall.denied), этап задержки и строка в Decisions.
// При запрете симуляция проваливается.
func (r *runner) firewallAllows(path []string, destinationIP string, port int) bool {
	sourceIP := ""
	if source, ok := r.nodes[r.req.Scenario.SourceNodeID]; ok {
		sourceIP = nodeIP(source)
	}

	for _, nodeID := range path {
		node := r.nodes[nodeID]
		if node.Type != model.NodeTypeFirewall {
			continue
		}

		start := r.timestamp
		processing := r.processingDelay(1, 5)
		r.advance(processing)

		allowed, decision := evaluateFirewall(node, sourceIP, destinationIP, httpsProtocol, port)

		eventType := model.EventFirewallDecision
		severity := model.EventSeverityInfo

		if !allowed {
			eventType = model.EventFirewallDenied
			severity = model.EventSeverityWarn
		}

		r.emit(eventType, severity, decision, r.req.Scenario.SourceNodeID, node.ID, r.packetID, map[string]any{
			"decision": decision,
			"allowed":  allowed,
		})
		r.addLatencyStage("firewall_decision", "Firewall decision", r.timestamp-start, map[string]any{
			"processingMs": processing,
			"allowed":      allowed,
			"decision":     decision,
			"nodeId":       node.ID,
		})
		r.summary.Decisions = append(r.summary.Decisions, decision)

		if !allowed {
			r.emit(model.EventPacketDropped, model.EventSeverityWarn, "packet dropped by firewall", r.req.Scenario.SourceNodeID, node.ID, r.packetID, nil)
			r.fail(decision)

			return false
		}

		return true
	}

	r.summary.Decisions = append(r.summary.Decisions, "No firewall on path")

	return true
}

// evaluateFirewall применяет правила firewall к пакету и возвращает решение
// и его объяснение.
//
// Правила проверяются по возрастанию priority, срабатывает первое подходящее
// по протоколу, порту, источнику и назначению; поле, которого в правиле нет,
// подходит к любому пакету. Не подошло ни одно — действует defaultPolicy
// (по умолчанию deny).
//
// Сортировка идёт на месте, в срезе из config узла: внутри запуска порядок
// правил после неё остаётся отсортированным.
func evaluateFirewall(node model.Node, sourceIP, destinationIP, protocol string, port int) (bool, string) {
	rules := anySlice(node.Config["rules"])
	sort.SliceStable(rules, func(i, j int) bool {
		return intValue(anyMap(rules[i])["priority"], defaultRulePriority) < intValue(anyMap(rules[j])["priority"], defaultRulePriority)
	})

	for _, item := range rules {
		rule := anyMap(item)

		if !ruleMatches(rule, sourceIP, destinationIP, protocol, port) {
			continue
		}

		action := defaultString(rule["action"], actionDeny)
		priority := intValue(rule["priority"], defaultRulePriority)

		return action == actionAllow, fmt.Sprintf("Firewall rule #%d %s tcp/%d from %s to %s", priority, strings.ToUpper(action), port, sourceIP, destinationIP)
	}

	defaultPolicy := defaultString(node.Config["defaultPolicy"], actionDeny)

	return defaultPolicy == actionAllow, "Firewall default policy " + strings.ToUpper(defaultPolicy)
}

// ruleMatches проверяет, подходит ли правило к пакету.
func ruleMatches(rule map[string]any, sourceIP, destinationIP, protocol string, port int) bool {
	return strings.EqualFold(defaultString(rule["protocol"], protocol), protocol) &&
		intValue(rule["port"], port) == port &&
		cidrMatches(defaultString(rule["source"], anyAddress), sourceIP) &&
		cidrMatches(defaultString(rule["destination"], anyAddress), destinationIP)
}

// cidrMatches — как cidrContains, но "", "any" и 0.0.0.0/0 подходят к любому
// адресу, даже пустому (у узла может не быть IP).
func cidrMatches(cidr, ipValue string) bool {
	if cidr == "" || cidr == "any" || cidr == anyAddress {
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
