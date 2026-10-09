// Package v1 — API-слой Quest Mode: /api/v1/quests/* и /api/v1/quest-attempts/*.
//
// Все маршруты — за middleware.RequireAuth. Старт квеста, проверки и успешные
// прохождения увеличивают счётчики quests_started_total, quest_checks_total
// и quests_completed_total.
package v1

import (
	"github.com/netquest/netquest/backend/internal/api/auditlog"
	"github.com/netquest/netquest/backend/internal/metrics"
)

// Имена параметров пути (ServeMux: {questId}, {attemptId}).
const (
	pathQuestID   = "questId"
	pathAttemptID = "attemptId"
)

// Ресурсы аудита.
const (
	auditResourceQuest   = "quest"
	auditResourceAttempt = "quest_attempt"
)

// api — HTTP-обработчики квестов.
type api struct {
	questService  QuestService
	auditRecorder auditlog.Recorder
	jsonLimit     int64
	metrics       *metrics.Metrics
}

// New создаёт обработчики квестов.
func New(questService QuestService, auditRecorder auditlog.Recorder, jsonLimit int64, counters *metrics.Metrics) *api {
	return &api{
		questService:  questService,
		auditRecorder: auditRecorder,
		jsonLimit:     jsonLimit,
		metrics:       counters,
	}
}
