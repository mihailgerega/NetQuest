package checker

import (
	"fmt"
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// Проверки, которые смотрят только на настройки узлов в документе,
// без запуска симуляции.

// highLatencyThresholdMs — с какой задержки канал считается медленным;
// тот же порог, что у замечания HIGH_LATENCY_LINK советника.
const highLatencyThresholdMs = 500

// checkDNS ищет в DNS-узлах A-запись Hostname и сравнивает её значение
// с ExpectedIP. Учитывается первая подходящая запись первого DNS-узла,
// в котором она есть; состояние DNS-узла не важно — проверяется настройка.
func checkDNS(doc model.Document, spec model.CheckSpec) model.CheckResult {
	for _, node := range doc.Nodes {
		if node.Type != model.NodeTypeDNS {
			continue
		}

		for _, record := range anySlice(node.Config["records"]) {
			m := anyMap(record)
			if !strings.EqualFold(stringValue(m["name"]), spec.Hostname) || !strings.EqualFold(defaultString(stringValue(m["type"]), "A"), "A") {
				continue
			}

			value := stringValue(m["value"])
			passed := value == spec.ExpectedIP

			message := "DNS record найден."
			if !passed {
				message = fmt.Sprintf("DNS record указывает на %s, ожидался %s.", value, spec.ExpectedIP)
			}

			return model.CheckResult{
				ID:      spec.ID,
				Passed:  passed,
				Message: message,
				Details: map[string]any{"nodeId": node.ID, "hostname": spec.Hostname, "value": value},
			}
		}
	}

	return model.CheckResult{
		ID:      spec.ID,
		Passed:  false,
		Message: "DNS record " + spec.Hostname + " отсутствует.",
		Details: map[string]any{"hostname": spec.Hostname, "expectedIp": spec.ExpectedIP},
	}
}

// checkFirewall находит правило firewall (NodeID или первого firewall),
// подходящее по протоколу, порту и назначению, и сравнивает его действие
// с ExpectedAction. Подходящего правила нет — сравнивается defaultPolicy.
//
// В отличие от движка, правила здесь не сортируются по priority:
// проверяется первое подходящее в порядке документа.
func checkFirewall(doc model.Document, spec model.CheckSpec) model.CheckResult {
	for _, node := range doc.Nodes {
		if node.Type != model.NodeTypeFirewall || (spec.NodeID != "" && node.ID != spec.NodeID) {
			continue
		}

		for _, raw := range anySlice(node.Config["rules"]) {
			rule := anyMap(raw)
			if !firewallRuleMatches(rule, spec) {
				continue
			}

			action := defaultString(stringValue(rule["action"]), "deny")
			passed := strings.EqualFold(action, spec.ExpectedAction)

			return model.CheckResult{
				ID:      spec.ID,
				Passed:  passed,
				Message: firewallMessage(passed, action, spec.ExpectedAction),
				Details: map[string]any{"nodeId": node.ID, "rule": rule},
			}
		}

		defaultPolicy := defaultString(stringValue(node.Config["defaultPolicy"]), "deny")
		passed := strings.EqualFold(defaultPolicy, spec.ExpectedAction)

		return model.CheckResult{
			ID:      spec.ID,
			Passed:  passed,
			Message: firewallMessage(passed, defaultPolicy, spec.ExpectedAction),
			Details: map[string]any{"nodeId": node.ID, "defaultPolicy": defaultPolicy},
		}
	}

	return model.CheckResult{ID: spec.ID, Passed: false, Message: "Firewall node не найден."}
}

// firewallRuleMatches сообщает, относится ли правило к проверке: протокол
// (по умолчанию tcp), порт (если задан в проверке) и назначение (если задан IP).
func firewallRuleMatches(rule map[string]any, spec model.CheckSpec) bool {
	if !strings.EqualFold(defaultString(stringValue(rule["protocol"]), "tcp"), defaultString(spec.Protocol, "tcp")) {
		return false
	}

	if spec.Port > 0 && intValue(rule["port"], 0) != spec.Port {
		return false
	}

	if spec.ExpectedIP != "" && !cidrOrIPMatches(defaultString(stringValue(rule["destination"]), "0.0.0.0/0"), spec.ExpectedIP) {
		return false
	}

	return true
}

// checkLB проверяет пул Load Balancer'а (NodeID или первого): пул не пуст,
// в нём все RequiredBackends и хотя бы один из AnyOfBackends.
// Исправность серверов здесь не проверяется — это дело сценарных проверок.
func checkLB(doc model.Document, spec model.CheckSpec) model.CheckResult {
	for _, node := range doc.Nodes {
		if node.Type != model.NodeTypeLoadBalancer || (spec.NodeID != "" && node.ID != spec.NodeID) {
			continue
		}

		return checkBackendPool(node, spec)
	}

	return model.CheckResult{ID: spec.ID, Passed: false, Message: "Load Balancer не найден."}
}

// checkBackendPool — проверка пула найденного Load Balancer'а.
func checkBackendPool(node model.Node, spec model.CheckSpec) model.CheckResult {
	backends := anySlice(node.Config["backends"])
	pool := make(map[string]bool, len(backends))

	for _, item := range backends {
		if id := stringValue(anyMap(item)["nodeId"]); id != "" {
			pool[id] = true
		}
	}

	if len(pool) == 0 {
		return model.CheckResult{ID: spec.ID, Passed: false, Message: "Load Balancer backend pool пустой.", Details: map[string]any{"nodeId": node.ID}}
	}

	for _, backendID := range spec.RequiredBackends {
		if !pool[backendID] {
			return model.CheckResult{
				ID:      spec.ID,
				Passed:  false,
				Message: "Backend pool не содержит " + backendID + ".",
				Details: map[string]any{"nodeId": node.ID, "pool": pool},
			}
		}
	}

	if len(spec.AnyOfBackends) > 0 && !poolHasAny(pool, spec.AnyOfBackends) {
		return model.CheckResult{
			ID:      spec.ID,
			Passed:  false,
			Message: "Backend pool не содержит ни один допустимый healthy backend.",
			Details: map[string]any{"nodeId": node.ID, "pool": pool, "anyOfBackends": spec.AnyOfBackends},
		}
	}

	return model.CheckResult{ID: spec.ID, Passed: true, Message: "Backend pool настроен.", Details: map[string]any{"nodeId": node.ID, "pool": pool}}
}

// poolHasAny сообщает, есть ли в пуле хотя бы один из серверов.
func poolHasAny(pool map[string]bool, backendIDs []string) bool {
	for _, backendID := range backendIDs {
		if pool[backendID] {
			return true
		}
	}

	return false
}

// checkAdvisorLike повторяет правило советника без запуска советника.
// Сейчас поддержано одно: HIGH_LATENCY_LINK — активного канала с задержкой
// от 500 мс быть не должно. Для остальных кодов проверка считается пройденной.
func checkAdvisorLike(doc model.Document, spec model.CheckSpec) model.CheckResult {
	if spec.ForbiddenIssueCode == "HIGH_LATENCY_LINK" {
		for _, link := range doc.Links {
			if intValue(link.Config["latencyMs"], 0) >= highLatencyThresholdMs && link.Status != "down" {
				return model.CheckResult{
					ID:      spec.ID,
					Passed:  false,
					Message: "В topology остался active link с высокой latency.",
					Details: map[string]any{"linkId": link.ID},
				}
			}
		}
	}

	return model.CheckResult{ID: spec.ID, Passed: true, Message: spec.Title + messageDoneSuffix}
}

// firewallMessage — сообщение проверки firewall.
func firewallMessage(passed bool, actual, expected string) string {
	if passed {
		return "Firewall rule настроен: " + expected + "."
	}

	return "Firewall action " + actual + ", ожидался " + expected + "."
}
