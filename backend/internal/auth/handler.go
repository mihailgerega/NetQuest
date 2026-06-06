package auth

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/netquest/netquest/backend/internal/audit"
	"github.com/netquest/netquest/backend/internal/httpx"
	"github.com/netquest/netquest/backend/pkg/apperrors"
)

type Handler struct {
	service      *Service
	auditService *audit.Service
	logger       *slog.Logger
	jsonLimit    int64
	secureCookie bool
}

func NewHandler(service *Service, auditService *audit.Service, logger *slog.Logger, jsonLimit int64, secureCookie bool) *Handler {
	return &Handler{service: service, auditService: auditService, logger: logger, jsonLimit: jsonLimit, secureCookie: secureCookie}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := httpx.DecodeJSON(r, &req, h.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	res, err := h.service.Register(r.Context(), req, r.UserAgent(), httpx.ClientIP(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.setRefreshCookie(w, res.RefreshToken)
	res.RefreshToken = ""
	h.recordAudit(r, &res.User.ID, "register", "user", res.User.ID)
	httpx.WriteJSON(w, http.StatusCreated, res)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpx.DecodeJSON(r, &req, h.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	res, err := h.service.Login(r.Context(), req, r.UserAgent(), httpx.ClientIP(r))
	if err != nil {
		h.recordAudit(r, nil, "failed_login", "user", "")
		httpx.WriteError(w, r, err)
		return
	}
	h.setRefreshCookie(w, res.RefreshToken)
	res.RefreshToken = ""
	h.recordAudit(r, &res.User.ID, "login", "user", res.User.ID)
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) Demo(w http.ResponseWriter, r *http.Request) {
	res, err := h.service.Demo(r.Context(), r.UserAgent(), httpx.ClientIP(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.setRefreshCookie(w, res.RefreshToken)
	res.RefreshToken = ""
	h.recordAudit(r, &res.User.ID, "login", "user", res.User.ID)
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	_ = httpx.DecodeJSON(r, &req, h.jsonLimit)
	if req.RefreshToken == "" {
		if cookie, err := r.Cookie(RefreshTokenCookieName); err == nil {
			req.RefreshToken = cookie.Value
		}
	}
	res, err := h.service.Refresh(r.Context(), req.RefreshToken, r.UserAgent(), httpx.ClientIP(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.setRefreshCookie(w, res.RefreshToken)
	res.RefreshToken = ""
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	_ = httpx.DecodeJSON(r, &req, h.jsonLimit)
	if req.RefreshToken == "" {
		if cookie, err := r.Cookie(RefreshTokenCookieName); err == nil {
			req.RefreshToken = cookie.Value
		}
	}
	if err := h.service.Logout(r.Context(), req.RefreshToken); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.clearRefreshCookie(w)
	var userID *string
	if principal, ok := PrincipalFromContext(r.Context()); ok {
		userID = &principal.UserID
	}
	h.recordAudit(r, userID, "logout", "user", "")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}
	user, err := h.service.Me(r.Context(), principal.UserID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"user": user,
	})
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) recordAudit(r *http.Request, userID *string, action, resourceType, resourceID string) {
	if h.auditService == nil {
		return
	}
	if err := h.auditService.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		IPAddress:    httpx.ClientIP(r),
		UserAgent:    r.UserAgent(),
	}); err != nil && h.logger != nil {
		h.logger.Warn("record audit log failed", slog.String("error", err.Error()))
	}
}
