package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/netquest/netquest/backend/internal/httpx"
	"github.com/netquest/netquest/backend/internal/storage"
)

type HealthHandler struct {
	ServiceName string
	Checker     storage.HealthChecker
	DeepTimeout time.Duration
}

func (h HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"status":    storage.HealthStatusOK,
		"service":   h.ServiceName,
		"timestamp": time.Now().UTC(),
	})
}

func (h HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	report := h.Checker.Ready(r.Context())
	status := http.StatusOK
	if report.Status == storage.HealthStatusError {
		status = http.StatusServiceUnavailable
	}
	httpx.WriteJSON(w, status, report)
}

func (h HealthHandler) Deep(w http.ResponseWriter, r *http.Request) {
	timeout := h.DeepTimeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	report := h.Checker.Deep(ctx)
	status := http.StatusOK
	if report.Status != storage.HealthStatusOK {
		status = http.StatusServiceUnavailable
	}
	httpx.WriteJSON(w, status, report)
}
