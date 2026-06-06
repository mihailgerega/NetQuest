package projects

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
	return &Handler{
		service:      service,
		auditService: auditService,
		logger:       logger,
		jsonLimit:    jsonLimit,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}

	projects, err := h.service.List(r.Context(), principal.UserID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
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

	project, err := h.service.Create(r.Context(), principal.UserID, req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	h.recordAudit(r, principal.UserID, "project.created", "project", project.ID)
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"project": project})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}

	project, err := h.service.Get(r.Context(), principal.UserID, r.PathValue("projectId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"project": project})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}

	var req UpdateRequest
	if err := httpx.DecodeJSON(r, &req, h.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	project, err := h.service.Update(r.Context(), principal.UserID, r.PathValue("projectId"), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"project": project})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}

	if err := h.service.Delete(r.Context(), principal.UserID, r.PathValue("projectId")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.recordAudit(r, principal.UserID, "project.deleted", "project", r.PathValue("projectId"))
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
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
