package model

import "time"

// Статусы health-check: и всего сервиса, и отдельной зависимости.
const (
	HealthStatusOK       = "ok"
	HealthStatusDegraded = "degraded" // PostgreSQL жив, но Redis или NATS — нет
	HealthStatusError    = "error"
)

// HealthReport — результат проверки зависимостей сервиса.
type HealthReport struct {
	Status    string
	Checks    map[string]ComponentCheck // ключ — имя зависимости: postgres, redis, nats
	Timestamp time.Time
}

// ComponentCheck — проверка одной зависимости.
type ComponentCheck struct {
	Status    string
	LatencyMs int64
	Error     string // пусто, если проверка прошла
}
