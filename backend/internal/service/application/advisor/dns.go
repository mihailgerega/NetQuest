package advisor

import (
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// apiHostname — домен, к которому обращаются сценарии и квесты NetQuest.
const apiHostname = "api.netquest.local"

// analyzeDNS: DNS-узел есть, но A-записи api.netquest.local нет ни в одном.
// Без DNS-узла правило молчит — значит, сценарий ходит по IP.
func analyzeDNS(doc model.Document) []model.Issue {
	hasDNS := false
	hasAPIRecord := false

	for _, node := range doc.Nodes {
		if node.Type != model.NodeTypeDNS {
			continue
		}

		hasDNS = true

		for _, record := range anySlice(node.Config["records"]) {
			m := anyMap(record)
			if strings.EqualFold(stringValue(m["name"]), apiHostname) && strings.EqualFold(defaultString(m["type"], "A"), "A") {
				hasAPIRecord = true
			}
		}
	}

	if !hasDNS || hasAPIRecord {
		return nil
	}

	return []model.Issue{{
		Severity:     model.IssueSeverityError,
		Category:     "DNS",
		Code:         "DNS_RECORD_MISSING",
		Title:        "DNS record отсутствует",
		Message:      "Домен api.netquest.local не найден в DNS records.",
		SuggestedFix: "Добавьте A record api.netquest.local → IP Load Balancer или Server.",
	}}
}
