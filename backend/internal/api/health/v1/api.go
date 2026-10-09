// Package v1 — API-слой health-check: /health/live, /health/ready, /health/deep.
//
// Маршруты без аутентификации: их зовут оркестратор, балансировщик и мониторинг.
//
//   - live — процесс жив (зависимости не проверяются): если он не отвечает,
//     контейнер пора перезапустить;
//   - ready — PING зависимостей: 503 только если недоступен PostgreSQL;
//   - deep — запись и чтение в каждой зависимости: 503 при любой проблеме.
package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/netquest/netquest/backend/internal/api/converter"
	healthv1 "github.com/netquest/netquest/backend/internal/contract/health/v1"
	"github.com/netquest/netquest/backend/internal/httpx"
	"github.com/netquest/netquest/backend/internal/model"
)

// defaultDeepTimeout — срок глубокой проверки, если он не задан в конфиге.
const defaultDeepTimeout = 3 * time.Second

// HealthChecker — то, что обработчикам нужно от сервиса проверок.
type HealthChecker interface {
	Ready(ctx context.Context) model.HealthReport
	Deep(ctx context.Context) model.HealthReport
}

// api — HTTP-обработчики health-check.
type api struct {
	serviceName string
	checker     HealthChecker
	deepTimeout time.Duration
}

// New создаёт обработчики health-check.
func New(serviceName string, checker HealthChecker, deepTimeout time.Duration) *api {
	return &api{
		serviceName: serviceName,
		checker:     checker,
		deepTimeout: deepTimeout,
	}
}

// Live обрабатывает GET /health/live. Ответ: всегда 200 {status: ok, service, timestamp}.
func (a *api) Live(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, healthv1.LiveResponse{
		Service:   a.serviceName,
		Status:    model.HealthStatusOK,
		Timestamp: time.Now().UTC(),
	})
}

// Ready обрабатывает GET /health/ready. Ответы: 200 (ok или degraded) / 503 (error).
func (a *api) Ready(w http.ResponseWriter, r *http.Request) {
	report := a.checker.Ready(r.Context())

	status := http.StatusOK
	if report.Status == model.HealthStatusError {
		status = http.StatusServiceUnavailable
	}

	httpx.WriteJSON(w, status, converter.HealthReportToDTO(report))
}

// Deep обрабатывает GET /health/deep. Ответы: 200 (ok) / 503 (иначе).
// У проверки свой срок, чтобы зависшая зависимость не держала запрос до таймаута сервера.
func (a *api) Deep(w http.ResponseWriter, r *http.Request) {
	timeout := a.deepTimeout
	if timeout <= 0 {
		timeout = defaultDeepTimeout
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	report := a.checker.Deep(ctx)

	status := http.StatusOK
	if report.Status != model.HealthStatusOK {
		status = http.StatusServiceUnavailable
	}

	httpx.WriteJSON(w, status, converter.HealthReportToDTO(report))
}
