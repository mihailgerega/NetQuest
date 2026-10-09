package v1

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/api/converter"
	"github.com/netquest/netquest/backend/internal/auth"
	advisorv1 "github.com/netquest/netquest/backend/internal/contract/advisor/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
)

// AnalyzeStored обрабатывает POST /api/v1/topologies/{topologyId}/analyze:
// анализ сохранённой версии.
// Ответы: 200 {issues} / 404 версии нет или проект чужой.
//
// Тело необязательно (в нём может быть только сценарий), поэтому ошибка его
// разбора игнорируется: анализ идёт без сценария.
func (a *api) AnalyzeStored(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	var req advisorv1.AnalyzeRequest

	_ = httpx.DecodeJSON(r, &req, a.jsonLimit) //nolint:gosec // G104: тело необязательно, см. комментарий к хендлеру

	topologyID := r.PathValue(pathTopologyID)

	issues, err := a.advisorService.AnalyzeStored(r.Context(), principal.UserID, topologyID, converter.OptionalScenarioToModel(req.Scenario))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	a.recordMetrics(len(issues))
	auditlog.Record(r, a.auditRecorder, &principal.UserID, "advisor_run", auditResourceTopology, topologyID)
	httpx.WriteJSON(w, http.StatusOK, advisorv1.AnalyzeResponse{Issues: issues})
}
