package advisor

import (
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// highLatencyThresholdMs — с какой задержки активный канал считается медленным.
const highLatencyThresholdMs = 500

// analyzeLatency: активный канал с задержкой от 500 мс заметно увеличит
// totalLatencyMs любого сценария, который через него пройдёт.
func analyzeLatency(doc model.Document) []model.Issue {
	issues := []model.Issue{}

	for _, link := range doc.Links {
		latency := intValue(link.Config["latencyMs"], 0)
		if link.Status == "down" || latency < highLatencyThresholdMs {
			continue
		}

		issues = append(issues, model.Issue{
			Severity:       model.IssueSeverityWarning,
			Category:       "Latency",
			Code:           "HIGH_LATENCY_LINK",
			Title:          "Высокая latency на link",
			Message:        fmt.Sprintf("Link %s → %s имеет latency %dms и может влиять на totalLatencyMs.", link.SourceNodeID, link.TargetNodeID, latency),
			AffectedLinkID: link.ID,
			SuggestedFix:   "Уменьшите latency или выберите другой route.",
		})
	}

	return issues
}
