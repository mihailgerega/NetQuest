package advisor

import "github.com/netquest/netquest/backend/internal/model"

// analyzeSource: клиент-источник сценария выключен — симуляция провалится
// на первом же шаге. Смотрится только поле status узла, без config.status.
func analyzeSource(doc model.Document, scenario *model.Scenario) []model.Issue {
	if scenario == nil || scenario.SourceNodeID == "" {
		return nil
	}

	for _, node := range doc.Nodes {
		if node.ID == scenario.SourceNodeID && node.Status == "down" {
			return []model.Issue{{
				Severity:       model.IssueSeverityError,
				Category:       "Topology",
				Code:           "SOURCE_CLIENT_DOWN",
				Title:          "Source Client down",
				Message:        "Выбранный Client недоступен.",
				AffectedNodeID: node.ID,
				SuggestedFix:   "Восстановите Client или выберите другой source.",
			}}
		}
	}

	return nil
}
