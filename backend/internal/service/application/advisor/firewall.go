package advisor

import (
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// httpsPort — порт HTTPS, который должен пропускать firewall.
const httpsPort = 443

// analyzeFirewalls: firewall с политикой deny по умолчанию, в котором нет
// ни одного правила allow для tcp/443, заблокирует HTTPS-сценарии.
//
// Это предупреждение, а не ошибка: порядок и адреса правил советник не
// разбирает, а топология может быть рассчитана не на HTTPS.
func analyzeFirewalls(doc model.Document) []model.Issue {
	issues := []model.Issue{}

	for _, node := range doc.Nodes {
		if node.Type != model.NodeTypeFirewall {
			continue
		}

		if !allowsHTTPS(node) && strings.EqualFold(defaultString(node.Config["defaultPolicy"], "deny"), "deny") {
			issues = append(issues, model.Issue{
				Severity:       model.IssueSeverityWarning,
				Category:       "Firewall",
				Code:           "FIREWALL_BLOCKS_HTTPS",
				Title:          "Firewall блокирует tcp/443",
				Message:        "Текущее правило Firewall может запрещать HTTPS traffic.",
				AffectedNodeID: node.ID,
				SuggestedFix:   "Добавьте allow rule для tcp/443 выше deny rule.",
			})
		}
	}

	return issues
}

// allowsHTTPS — есть ли у firewall правило allow для tcp/443
// (протокол по умолчанию tcp, действие по умолчанию deny).
func allowsHTTPS(node model.Node) bool {
	for _, raw := range anySlice(node.Config["rules"]) {
		rule := anyMap(raw)

		if strings.EqualFold(defaultString(rule["protocol"], "tcp"), "tcp") &&
			intValue(rule["port"], 0) == httpsPort &&
			strings.EqualFold(defaultString(rule["action"], "deny"), "allow") {
			return true
		}
	}

	return false
}
