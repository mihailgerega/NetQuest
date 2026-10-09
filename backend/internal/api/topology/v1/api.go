// Package v1 — API-слой версий топологии:
// /api/v1/projects/{projectId}/topologies и /api/v1/topologies/{topologyId}.
//
// Все маршруты — за middleware.RequireAuth. Поток запроса — как у auth/v1:
// DecodeJSON → converter → service → converter → WriteJSON, ошибки — httpx.WriteError.
package v1

import "github.com/netquest/netquest/backend/internal/api/auditlog"

// Имена параметров пути (ServeMux: {projectId}, {topologyId}).
const (
	pathProjectID  = "projectId"
	pathTopologyID = "topologyId"
)

// api — HTTP-обработчики версий топологии.
type api struct {
	topologyService TopologyService
	auditRecorder   auditlog.Recorder
	jsonLimit       int64
}

// New создаёт обработчики версий топологии.
func New(topologyService TopologyService, auditRecorder auditlog.Recorder, jsonLimit int64) *api {
	return &api{
		topologyService: topologyService,
		auditRecorder:   auditRecorder,
		jsonLimit:       jsonLimit,
	}
}
