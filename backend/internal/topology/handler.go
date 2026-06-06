package topology

import (
	"log/slog"
	"net/http"

	"github.com/netquest/netquest/backend/internal/audit"
	"github.com/netquest/netquest/backend/internal/auth"
	"github.com/netquest/netquest/backend/internal/httpx"
	"github.com/netquest/netquest/backend/pkg/apperrors"
)

type Handler struct {
	service      *Service
	auditService *audit.Service
	logger       *slog.Logger
	jsonLimit    int64
}

func NewHandler(service *Service, auditService *audit.Service, logger *slog.Logger, jsonLimit int64) *Handler {
	return &Handler{service: service, auditService: auditService, logger: logger, jsonLimit: jsonLimit}
}

func (h *Handler) ListForProject(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}

	items, err := h.service.ListForProject(r.Context(), principal.UserID, r.PathValue("projectId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"topologies": items})
}

func (h *Handler) CreateForProject(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}

	var req CreateRequest
	if err := httpx.DecodeJSON(r, &req, h.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	result, err := h.service.Create(r.Context(), principal.UserID, r.PathValue("projectId"), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	h.recordAudit(r, principal.UserID, "topology.created", "topology", result.Topology.ID)
	httpx.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}

	topology, err := h.service.Get(r.Context(), principal.UserID, r.PathValue("topologyId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"topology": topology})
}

func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}

	result, err := h.service.ValidateStored(r.Context(), principal.UserID, r.PathValue("topologyId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, result)
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
		h.logger.Warn("record audit log failed", slog.String("error", err.Error()))
	}
}
