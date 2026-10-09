// Package advisorv1 — HTTP-контракт Validation Advisor:
// /api/v1/topologies/analyze и /api/v1/topologies/{topologyId}/analyze.
//
// Имена типов запросов значимы: encoding/json вставляет их в текст ошибки
// разбора, который уходит клиенту в ответе 400.
package advisorv1

import (
	"encoding/json"

	simulationv1 "github.com/netquest/netquest/backend/internal/contract/simulation/v1"
	"github.com/netquest/netquest/backend/internal/model"
)

// AnalyzeRequest — тело запроса анализа. Для сохранённой версии Topology
// не нужен (и тело целиком необязательно); сценарий уточняет правила
// маршрутизации и источника.
type AnalyzeRequest struct {
	Topology json.RawMessage        `json:"topology,omitempty"`
	Scenario *simulationv1.Scenario `json:"scenario,omitempty"`
}

// AnalyzeResponse — замечания советника.
type AnalyzeResponse struct {
	Issues []model.Issue `json:"issues"`
}
