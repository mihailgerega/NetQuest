package advisor

import (
	"encoding/json"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
)

// AnalyzeRaw диагностирует топологию, присланную в запросе (ещё не сохранённую).
//
// Сначала — ошибки валидации, каждая отдельным замечанием TOPOLOGY_INVALID.
// Они не останавливают анализ: правила советника работают и на частично
// сломанном документе, лишь бы он разбирался как JSON. Затем правила по слоям:
// DNS → Load Balancer → firewall → задержка → маршрутизация → источник.
// Сценарий (может быть nil) уточняет правила маршрутизации и источника.
func (s *service) AnalyzeRaw(data json.RawMessage, scenario *model.Scenario) ([]model.Issue, error) {
	if len(data) == 0 {
		return nil, errs.NewValidationError("topology is required", nil)
	}

	issues := []model.Issue{}

	validation := s.validator.ValidateRaw(data)
	if !validation.Valid {
		for _, item := range validation.Errors {
			issues = append(issues, model.Issue{
				Severity:     model.IssueSeverityError,
				Category:     "Topology",
				Code:         "TOPOLOGY_INVALID",
				Title:        "Topology не проходит validation",
				Message:      item.Path + ": " + item.Message,
				SuggestedFix: "Исправьте структуру topology перед запуском simulation.",
			})
		}
	}

	var doc model.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, errs.NewValidationError("invalid topology JSON", err.Error())
	}

	issues = append(issues, analyzeDNS(doc)...)
	issues = append(issues, analyzeLoadBalancers(doc)...)
	issues = append(issues, analyzeFirewalls(doc)...)
	issues = append(issues, analyzeLatency(doc)...)
	issues = append(issues, analyzeRouting(doc, scenario)...)
	issues = append(issues, analyzeSource(doc, scenario)...)

	return issues, nil
}
