package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	advisorv1 "github.com/netquest/netquest/backend/internal/contract/advisor/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// AnalyzeRaw обрабатывает POST /api/v1/topologies/analyze: анализ топологии
// из тела запроса (ещё не сохранённой — фронтенд зовёт его прямо при правке).
// Ответы: 200 {issues} / 400 кривой JSON / 422 нет топологии или она не JSON-объект.
func (a *api) AnalyzeRaw(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	var req advisorv1.AnalyzeRequest
	if err := httpx.DecodeJSON(r, &req, a.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	issues, err := a.advisorService.AnalyzeRaw(req.Topology, converter.OptionalScenarioToModel(req.Scenario))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	a.recordMetrics(len(issues))
	auditlog.Record(r, a.auditRecorder, &principal.UserID, "topology_analyzed", auditResourceTopology, "inline")
	httpx.WriteJSON(w, http.StatusOK, advisorv1.AnalyzeResponse{Issues: issues})
}
