// Package v1 — API-слой проектов: обработчики /api/v1/projects/*.
//
// Все маршруты — за middleware.RequireAuth: хендлер берёт пользователя из ctx
// и работает только с его проектами. Поток запроса — как у auth/v1:
// DecodeJSON → converter → service → converter → WriteJSON, ошибки — httpx.WriteError.
package v1

import "github.com/netquest/netquest/backend/internal/api/auditlog"

// Действия и ресурс аудита.
const (
	auditActionCreated  = "project.created"
	auditActionDeleted  = "project.deleted"
	auditResourceObject = "project"
)

// pathProjectID — имя параметра пути с ID проекта (ServeMux: {projectId}).
const pathProjectID = "projectId"

// api — HTTP-обработчики проектов.
type api struct {
	projectService ProjectService
	auditRecorder  auditlog.Recorder
	jsonLimit      int64
}

// New создаёт обработчики проектов.
func New(projectService ProjectService, auditRecorder auditlog.Recorder, jsonLimit int64) *api {
	return &api{
		projectService: projectService,
		auditRecorder:  auditRecorder,
		jsonLimit:      jsonLimit,
	}
}
