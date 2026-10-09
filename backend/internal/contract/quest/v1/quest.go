// Package questv1 — HTTP-контракт Quest Mode: /api/v1/quests/* и /api/v1/quest-attempts/*.
//
// Квест и результат проверки — доменные модели с json-тегами (model.Quest,
// model.QuestResult): их JSON-форма общая для API и jsonb-колонок.
// Имена типов запросов значимы: encoding/json вставляет их в текст ошибки
// разбора, который уходит клиенту в ответе 400.
package questv1

import (
	"encoding/json"
	"time"

	"github.com/netquest/netquest/backend/internal/model"
)

// CheckRequest — тело POST /api/v1/quest-attempts/{attemptId}/check.
type CheckRequest struct {
	Topology json.RawMessage `json:"topology"`
	Seed     *int64          `json:"seed,omitempty"`
}

// RevealHintRequest — тело POST /api/v1/quest-attempts/{attemptId}/reveal-hint:
// сколько подсказок пользователь открыл всего.
type RevealHintRequest struct {
	RevealedHintsCount int `json:"revealedHintsCount"`
}

// Attempt — попытка в ответах. LastCheckResult — сохранённый QuestResult как есть.
type Attempt struct {
	ID                 string              `json:"id"`
	QuestID            string              `json:"questId"`
	UserID             string              `json:"userId,omitempty"`
	ProjectID          *string             `json:"projectId,omitempty"`
	CurrentTopologyID  *string             `json:"currentTopologyId,omitempty"`
	Status             model.AttemptStatus `json:"status"`
	AttemptsCount      int                 `json:"attemptsCount"`
	RevealedHintsCount int                 `json:"revealedHintsCount"`
	LastCheckResult    json.RawMessage     `json:"lastCheckResult,omitempty"`
	CompletedAt        *time.Time          `json:"completedAt,omitempty"`
	CreatedAt          time.Time           `json:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt"`
}

// ListResponse — ответ GET /api/v1/quests.
type ListResponse struct {
	Quests []model.Quest `json:"quests"`
}

// QuestResponse — ответ GET /api/v1/quests/{questId}.
type QuestResponse struct {
	Quest model.Quest `json:"quest"`
}

// QuestAttemptResponse — ответ старта и сброса: квест (со статусом попытки)
// и сама попытка.
type QuestAttemptResponse struct {
	Quest   model.Quest `json:"quest"`
	Attempt Attempt     `json:"attempt"`
}

// AttemptResponse — ответ с одной попыткой.
type AttemptResponse struct {
	Attempt Attempt `json:"attempt"`
}

// CheckResponse — ответ проверки: обновлённая попытка и результат проверки.
type CheckResponse struct {
	Attempt Attempt           `json:"attempt"`
	Result  model.QuestResult `json:"result"`
}
