// Package metrics — счётчики NetQuest API в текстовом формате Prometheus.
//
// Без клиентской библиотеки Prometheus: счётчиков немного, и все они —
// монотонные int64, которые хватает atomic.Int64. GET /metrics печатает их
// в формате exposition («имя значение» построчно), его понимает Prometheus.
//
// Один экземпляр Metrics создаёт DI-контейнер; в него пишут middleware
// (HTTP-запросы), сервис симуляций и API-хендлеры квестов, советника и WebSocket.
package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// Metrics — счётчики сервиса. atomic.Int64 безопасен для параллельных
// горутин-запросов без мьютекса: каждое Add — одна атомарная инструкция.
type Metrics struct {
	HTTPRequestsTotal          atomic.Int64
	HTTPRequestDurationMsTotal atomic.Int64 // сумма длительностей: среднее = сумма / число запросов
	SimulationsStartedTotal    atomic.Int64
	SimulationsCompletedTotal  atomic.Int64
	SimulationsFailedTotal     atomic.Int64
	QuestsStartedTotal         atomic.Int64
	QuestsCompletedTotal       atomic.Int64
	QuestChecksTotal           atomic.Int64
	AdvisorRunsTotal           atomic.Int64
	AdvisorIssuesTotal         atomic.Int64
	RouteLookupFailuresTotal   atomic.Int64 // зарезервирован: пока ничем не увеличивается
	ActiveWebSocketConnections atomic.Int64 // gauge: растёт при подключении, падает при отключении
}

// New создаёт счётчики с нулями.
func New() *Metrics {
	return &Metrics{}
}

// Middleware считает HTTP-запросы и их суммарную длительность.
// Стоит в цепочке последним — меряет время самого обработчика.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		m.HTTPRequestsTotal.Add(1)
		m.HTTPRequestDurationMsTotal.Add(time.Since(start).Milliseconds())
	})
}

// Handler отдаёт счётчики в текстовом формате Prometheus. Ошибки записи
// игнорируются: клиент оборвал соединение — отдавать уже некому.
func (m *Metrics) Handler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	counters := []struct {
		name  string
		value int64
	}{
		{"http_requests_total", m.HTTPRequestsTotal.Load()},
		{"http_request_duration_ms_total", m.HTTPRequestDurationMsTotal.Load()},
		{"simulations_started_total", m.SimulationsStartedTotal.Load()},
		{"simulations_completed_total", m.SimulationsCompletedTotal.Load()},
		{"simulations_failed_total", m.SimulationsFailedTotal.Load()},
		{"quests_started_total", m.QuestsStartedTotal.Load()},
		{"quests_completed_total", m.QuestsCompletedTotal.Load()},
		{"quest_checks_total", m.QuestChecksTotal.Load()},
		{"advisor_runs_total", m.AdvisorRunsTotal.Load()},
		{"advisor_issues_total", m.AdvisorIssuesTotal.Load()},
		{"route_lookup_failures_total", m.RouteLookupFailuresTotal.Load()},
		{"active_websocket_connections", m.ActiveWebSocketConnections.Load()},
	}

	for _, counter := range counters {
		_, _ = fmt.Fprintf(w, "%s %d\n", counter.name, counter.value)
	}
}
