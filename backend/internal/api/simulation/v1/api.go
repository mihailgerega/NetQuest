// Package v1 — API-слой симуляций: запуск и чтение (/api/v1/simulations/*)
// и поток событий по WebSocket (/api/v1/ws).
//
// REST-маршруты — за middleware.RequireAuth. WebSocket — нет: браузерный
// WebSocket не умеет ставить заголовок Authorization, поэтому токен приходит
// в query-параметре ?token= и проверяется в самом хендлере (stream.go).
package v1

import (
	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/metrics"
)

// pathSimulationID — имя параметра пути с ID симуляции.
const pathSimulationID = "simulationId"

// api — HTTP-обработчики симуляций.
type api struct {
	simulationService SimulationService
	auditRecorder     auditlog.Recorder
	tokenParser       AccessTokenParser
	jsonLimit         int64
	metrics           *metrics.Metrics
}

// New создаёт обработчики симуляций.
func New(
	simulationService SimulationService,
	auditRecorder auditlog.Recorder,
	tokenParser AccessTokenParser,
	jsonLimit int64,
	counters *metrics.Metrics,
) *api {
	return &api{
		simulationService: simulationService,
		auditRecorder:     auditRecorder,
		tokenParser:       tokenParser,
		jsonLimit:         jsonLimit,
		metrics:           counters,
	}
}
