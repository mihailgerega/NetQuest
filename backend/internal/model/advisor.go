package model

// IssueSeverity — важность замечания советника (Validation Advisor).
type IssueSeverity string

// Значения важности. Набор отличается от EventSeverity ("warning", а не "warn"):
// это разные словари, и фронтенд различает их.
const (
	IssueSeverityInfo    IssueSeverity = "info"
	IssueSeverityWarning IssueSeverity = "warning"
	IssueSeverityError   IssueSeverity = "error"
)

// Issue — замечание советника к топологии до запуска симуляции:
// что не так, где (узел или канал) и как исправить.
type Issue struct {
	Severity       IssueSeverity `json:"severity"`
	Category       string        `json:"category"`
	Code           string        `json:"code"`
	Title          string        `json:"title"`
	Message        string        `json:"message"`
	AffectedNodeID string        `json:"affectedNodeId,omitempty"`
	AffectedLinkID string        `json:"affectedLinkId,omitempty"`
	SuggestedFix   string        `json:"suggestedFix"`
}
