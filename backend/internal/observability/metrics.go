package observability

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

type Metrics struct {
	HTTPRequestsTotal          atomic.Int64
	HTTPRequestDurationMsTotal atomic.Int64
	SimulationsStartedTotal    atomic.Int64
	SimulationsCompletedTotal  atomic.Int64
	SimulationsFailedTotal     atomic.Int64
	QuestsStartedTotal         atomic.Int64
	QuestsCompletedTotal       atomic.Int64
	QuestChecksTotal           atomic.Int64
	AdvisorRunsTotal           atomic.Int64
	AdvisorIssuesTotal         atomic.Int64
	RouteLookupFailuresTotal   atomic.Int64
	ActiveWebSocketConnections atomic.Int64
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		m.HTTPRequestsTotal.Add(1)
		m.HTTPRequestDurationMsTotal.Add(time.Since(start).Milliseconds())
	})
}

func (m *Metrics) Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = fmt.Fprintf(w, "http_requests_total %d\n", m.HTTPRequestsTotal.Load())
	_, _ = fmt.Fprintf(w, "http_request_duration_ms_total %d\n", m.HTTPRequestDurationMsTotal.Load())
	_, _ = fmt.Fprintf(w, "simulations_started_total %d\n", m.SimulationsStartedTotal.Load())
	_, _ = fmt.Fprintf(w, "simulations_completed_total %d\n", m.SimulationsCompletedTotal.Load())
	_, _ = fmt.Fprintf(w, "simulations_failed_total %d\n", m.SimulationsFailedTotal.Load())
	_, _ = fmt.Fprintf(w, "quests_started_total %d\n", m.QuestsStartedTotal.Load())
	_, _ = fmt.Fprintf(w, "quests_completed_total %d\n", m.QuestsCompletedTotal.Load())
	_, _ = fmt.Fprintf(w, "quest_checks_total %d\n", m.QuestChecksTotal.Load())
	_, _ = fmt.Fprintf(w, "advisor_runs_total %d\n", m.AdvisorRunsTotal.Load())
	_, _ = fmt.Fprintf(w, "advisor_issues_total %d\n", m.AdvisorIssuesTotal.Load())
	_, _ = fmt.Fprintf(w, "route_lookup_failures_total %d\n", m.RouteLookupFailuresTotal.Load())
	_, _ = fmt.Fprintf(w, "active_websocket_connections %d\n", m.ActiveWebSocketConnections.Load())
}
