package v1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	simulationv1 "github.com/netquest/netquest/backend/internal/contract/simulation/v1"
	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/httpx"
)

const (
	// streamMessageType — тип кадра с событием симуляции.
	streamMessageType = "simulation.event"
	// streamEventInterval — пауза между кадрами: Timeline на фронтенде
	// проигрывает события по одному, а не показывает все разом.
	streamEventInterval = 40 * time.Millisecond
)

// Stream обрабатывает GET /api/v1/ws?simulationId=...&token=... — проигрывает
// сохранённые события симуляции по WebSocket и закрывает соединение.
//
// До upgrade ошибки — обычные HTTP-ответы: 401 без действительного токена,
// 422 без simulationId, 404 для чужой симуляции. После upgrade соединение
// принадлежит хендлеру, и при сбое записи он просто его закрывает.
//
// Токен берётся из ?token=, а без него — из Authorization (Bearer).
func (a *api) Stream(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}

	principal, err := a.tokenParser.ParseAccessToken(token)
	if err != nil {
		httpx.WriteError(w, r, errs.ErrWebSocketTokenInvalid)
		return
	}

	simulationID := strings.TrimSpace(r.URL.Query().Get("simulationId"))
	if simulationID == "" {
		httpx.WriteError(w, r, errs.NewValidationError("simulationId is required", nil))
		return
	}

	if _, err := a.simulationService.Get(r.Context(), principal.UserID, simulationID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	events, err := a.simulationService.Events(r.Context(), principal.UserID, simulationID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	conn, err := upgradeWebSocket(w, r)
	if err != nil {
		slog.WarnContext(r.Context(), "не удалось переключить соединение на WebSocket", "error", err)
		return
	}
	defer conn.Close()

	a.metrics.ActiveWebSocketConnections.Add(1)
	defer a.metrics.ActiveWebSocketConnections.Add(-1)

	for _, event := range events {
		payload, err := json.Marshal(simulationv1.StreamMessage{
			Event:        event,
			SimulationID: simulationID,
			Type:         streamMessageType,
		})
		if err != nil {
			return
		}

		if err := writeWebSocketText(conn, payload); err != nil {
			return
		}

		// Пауза без ctx запроса: после hijack соединение живёт само по себе,
		// а срок ctx (middleware.RequestTimeout) оборвал бы длинный Timeline.
		time.Sleep(streamEventInterval) //nolint:forbidigo // темп проигрывания событий, отменять нечего
	}

	_ = writeWebSocketClose(conn) //nolint:gosec // G104: кадр закрытия — вежливость; клиент мог уже уйти
}
