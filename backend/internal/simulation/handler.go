package simulation

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

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
	jwt          *auth.JWTManager
	metrics      *observability.Metrics
}

func NewHandler(service *Service, auditService *audit.Service, logger *slog.Logger, jsonLimit int64, jwt *auth.JWTManager, metrics *observability.Metrics) *Handler {
	return &Handler{service: service, auditService: auditService, logger: logger, jsonLimit: jsonLimit, jwt: jwt, metrics: metrics}
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}

	var req StartRequest
	if err := httpx.DecodeJSON(r, &req, h.jsonLimit); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	result, err := h.service.Start(r.Context(), principal.UserID, req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	h.recordAudit(r, principal.UserID, "simulation.started", "simulation", result.Simulation.ID)
	httpx.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}
	simulation, err := h.service.Get(r.Context(), principal.UserID, r.PathValue("simulationId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"simulation": simulation})
}

func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperrors.Unauthorized("authentication is required"))
		return
	}
	events, err := h.service.Events(r.Context(), principal.UserID, r.PathValue("simulationId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	if h.jwt == nil {
		httpx.WriteError(w, r, apperrors.Internal("websocket auth is not configured"))
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	principal, err := h.jwt.ParseAccessToken(token)
	if err != nil {
		httpx.WriteError(w, r, apperrors.Unauthorized("websocket token is invalid"))
		return
	}

	simulationID := strings.TrimSpace(r.URL.Query().Get("simulationId"))
	if simulationID == "" {
		httpx.WriteError(w, r, apperrors.Validation("simulationId is required", nil))
		return
	}
	if _, err := h.service.Get(r.Context(), principal.UserID, simulationID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	events, err := h.service.Events(r.Context(), principal.UserID, simulationID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	conn, err := upgradeWebSocket(w, r)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("websocket upgrade failed", slog.String("error", err.Error()))
		}
		return
	}
	defer conn.Close()
	if h.metrics != nil {
		h.metrics.ActiveWebSocketConnections.Add(1)
		defer h.metrics.ActiveWebSocketConnections.Add(-1)
	}

	for _, event := range events {
		payload, err := json.Marshal(map[string]any{
			"type":         "simulation.event",
			"simulationId": simulationID,
			"event":        event,
		})
		if err != nil {
			return
		}
		if err := writeWebSocketText(conn, payload); err != nil {
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
	_ = writeWebSocketClose(conn)
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

func upgradeWebSocket(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "websocket upgrade required", http.StatusBadRequest)
		return nil, fmt.Errorf("websocket upgrade required")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "websocket key required", http.StatusBadRequest)
		return nil, fmt.Errorf("websocket key required")
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket hijack unsupported", http.StatusInternalServerError)
		return nil, fmt.Errorf("websocket hijack unsupported")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", websocketAccept(key))
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func websocketAccept(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func writeWebSocketText(conn net.Conn, payload []byte) error {
	return writeWebSocketFrame(conn, 0x1, payload)
}

func writeWebSocketClose(conn net.Conn) error {
	return writeWebSocketFrame(conn, 0x8, nil)
}

func writeWebSocketFrame(conn net.Conn, opcode byte, payload []byte) error {
	header := []byte{0x80 | opcode}
	length := len(payload)
	switch {
	case length < 126:
		header = append(header, byte(length))
	case length <= 65535:
		header = append(header, 126, byte(length>>8), byte(length))
	default:
		header = append(header, 127, byte(length>>56), byte(length>>48), byte(length>>40), byte(length>>32), byte(length>>24), byte(length>>16), byte(length>>8), byte(length))
	}
	if _, err := conn.Write(header); err != nil {
		return err
	}
	_, err := conn.Write(payload)
	return err
}
