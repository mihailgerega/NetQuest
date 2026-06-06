package quests

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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}
	quests, err := h.service.List(r.Context(), principal.UserID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ListResponse{Quests: quests})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}
	quest, err := h.service.Get(r.Context(), principal.UserID, r.PathValue("questId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"quest": quest})
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}
	result, err := h.service.Start(r.Context(), principal.UserID, r.PathValue("questId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if h.metrics != nil {
		h.metrics.QuestsStartedTotal.Add(1)
	}
	h.recordAudit(r, principal.UserID, "quest_started", "quest", result.Quest.ID)
	httpx.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) GetAttempt(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}
	attempt, err := h.service.GetAttempt(r.Context(), principal.UserID, r.PathValue("attemptId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"attempt": attempt})
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}
	var req CheckRequest
	if err := httpx.DecodeJSON(r, &req, h.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	result, err := h.service.Check(r.Context(), principal.UserID, r.PathValue("attemptId"), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if h.metrics != nil {
		h.metrics.QuestChecksTotal.Add(1)
		if result.Result.Passed {
			h.metrics.QuestsCompletedTotal.Add(1)
		}
	}
	action := "quest_checked"
	if result.Result.Passed {
		action = "quest_completed"
	}
	h.recordAudit(r, principal.UserID, action, "quest_attempt", result.Attempt.ID)
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Reset(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}
	result, err := h.service.Reset(r.Context(), principal.UserID, r.PathValue("attemptId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.recordAudit(r, principal.UserID, "quest_reset", "quest_attempt", result.Attempt.ID)
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) RevealHint(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}
	var req RevealHintRequest
	if err := httpx.DecodeJSON(r, &req, h.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	result, err := h.service.RevealHint(r.Context(), principal.UserID, r.PathValue("attemptId"), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.recordAudit(r, principal.UserID, "quest_hint_revealed", "quest_attempt", result.Attempt.ID)
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
		h.logger.Warn("record quest audit log failed", slog.String("error", err.Error()))
	}
}
