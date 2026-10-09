// Package v1 — API-слой Validation Advisor: анализ присланной топологии
// (POST /api/v1/topologies/analyze) и сохранённой версии
// (POST /api/v1/topologies/{topologyId}/analyze).
//
// Оба маршрута — за middleware.RequireAuth. Каждый успешный анализ
// увеличивает счётчики advisor_runs_total и advisor_issues_total.
package v1

import (
	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/metrics"
)

// pathTopologyID — имя параметра пути с ID версии топологии.
const pathTopologyID = "topologyId"

// auditResourceTopology — ресурс аудита для анализа.
const auditResourceTopology = "topology"

// api — HTTP-обработчики советника.
type api struct {
	advisorService AdvisorService
	auditRecorder  auditlog.Recorder
	jsonLimit      int64
	metrics        *metrics.Metrics
}

// New создаёт обработчики советника.
func New(advisorService AdvisorService, auditRecorder auditlog.Recorder, jsonLimit int64, counters *metrics.Metrics) *api {
	return &api{
		advisorService: advisorService,
		auditRecorder:  auditRecorder,
		jsonLimit:      jsonLimit,
		metrics:        counters,
	}
}

// recordMetrics считает запуск советника и найденные замечания.
func (a *api) recordMetrics(issueCount int) {
	a.metrics.AdvisorRunsTotal.Add(1)
	a.metrics.AdvisorIssuesTotal.Add(int64(issueCount))
}
