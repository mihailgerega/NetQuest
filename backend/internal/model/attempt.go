package model

import (
	"encoding/json"
	"time"
)

// AttemptStatus — состояние попытки прохождения квеста.
type AttemptStatus string

// not_started не хранится в базе: его сервис подставляет квестам,
// у которых у пользователя ещё нет ни одной попытки.
const (
	AttemptNotStarted AttemptStatus = "not_started"
	AttemptInProgress AttemptStatus = "in_progress"
	AttemptCompleted  AttemptStatus = "completed"
	AttemptFailed     AttemptStatus = "failed"
)

// Attempt — попытка пользователя пройти квест.
//
// LastCheckResult — QuestResult последней проверки как сырой JSON: сервис его
// не разбирает, только сохраняет и отдаёт. RevealedHintsCount — сколько
// прогрессивных подсказок уже открыто; со временем только растёт.
type Attempt struct {
	ID                 string
	QuestID            string
	UserID             string
	ProjectID          *string // nil — попытка не привязана к проекту
	CurrentTopologyID  *string // nil — топология попытки не сохранялась
	Status             AttemptStatus
	AttemptsCount      int
	RevealedHintsCount int
	LastCheckResult    json.RawMessage // nil — проверок ещё не было
	CompletedAt        *time.Time      // nil — квест не пройден
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
