package advisor

import (
	"encoding/json"

	"github.com/netquest/netquest/backend/internal/simulation"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Issue struct {
	Severity       Severity `json:"severity"`
	Category       string   `json:"category"`
	Code           string   `json:"code"`
	Title          string   `json:"title"`
	Message        string   `json:"message"`
	AffectedNodeID string   `json:"affectedNodeId,omitempty"`
	AffectedLinkID string   `json:"affectedLinkId,omitempty"`
	SuggestedFix   string   `json:"suggestedFix"`
}

type AnalyzeRequest struct {
	Topology json.RawMessage      `json:"topology,omitempty"`
	Scenario *simulation.Scenario `json:"scenario,omitempty"`
}

type AnalyzeResponse struct {
	Issues []Issue `json:"issues"`
}
