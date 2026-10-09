package v1

import (
	"context"
	"encoding/json"

	"github.com/netquest/netquest/backend/internal/model"
)

// AdvisorService — то, что обработчикам нужно от советника.
// Реализация — *service из service/application/advisor. Мок: mocks/ (task mocks:gen).
type AdvisorService interface {
	AnalyzeRaw(data json.RawMessage, scenario *model.Scenario) ([]model.Issue, error)
	AnalyzeStored(ctx context.Context, userID, topologyID string, scenario *model.Scenario) ([]model.Issue, error)
}
