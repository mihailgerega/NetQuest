package advisor

import (
	"log/slog"
	"net/http"

	"github.com/netquest/netquest/backend/internal/audit"
	"github.com/netquest/netquest/backend/internal/auth"
	"github.com/netquest/netquest/backend/internal/httpx"
	"github.com/netquest/netquest/backend/internal/observability"
	"github.com/netquest/netquest/backend/pkg/apperrors"
)

type Handler struct {
	service      *Service
	auditService *audit.Service
	logger       *slog.Logger
	jsonLimit    int64
	metrics      *observability.Metrics
}

func NewHandler(service *Service, auditService *audit.Service, logger *slog.Logger, jsonLimit int64, metrics *observability.Metrics) *Handler {
	return &Handler{service: service, auditService: auditService, logger: logger, jsonLimit: jsonLimit, metrics: metrics}
}

func (h *Handler) AnalyzeRaw(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}
	var req AnalyzeRequest
	if err := httpx.DecodeJSON(r, &req, h.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	res, err := h.service.AnalyzeRaw(req.Topology, req.Scenario)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.recordMetrics(len(res.Issues))
	h.recordAudit(r, principal.UserID, "topology_analyzed", "topology", "inline")
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) AnalyzeStored(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}
	var req AnalyzeRequest
	if r.Body != nil {
		_ = httpx.DecodeJSON(r, &req, h.jsonLimit)
	}
	res, err := h.service.AnalyzeStored(r.Context(), principal.UserID, r.PathValue("topologyId"), req.Scenario)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.recordMetrics(len(res.Issues))
	h.recordAudit(r, principal.UserID, "advisor_run", "topology", r.PathValue("topologyId"))
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) recordMetrics(issueCount int) {
	if h.metrics == nil {
		return
	}
	h.metrics.AdvisorRunsTotal.Add(1)
	h.metrics.AdvisorIssuesTotal.Add(int64(issueCount))
}

func (h *Handler) recordAudit(r *http.Request, userID, action, resourceType, resourceID string) {
	if h.auditService == nil {
		return
	}
	if err := h.auditService.Record(r.Context(), audit.Entry{
		UserID:       &userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		IPAddress:    httpx.ClientIP(r),
		UserAgent:    r.UserAgent(),
	}); err != nil && h.logger != nil {
		h.logger.Warn("record advisor audit log failed", slog.String("error", err.Error()))
	}
}
