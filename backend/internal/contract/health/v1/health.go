// Package healthv1 — HTTP-контракт health-check: /health/live, /health/ready, /health/deep.
package healthv1

import "time"

// LiveResponse — ответ /health/live: процесс жив и отвечает.
// Поля в алфавитном порядке, как в прежнем ответе (map[string]any).
type LiveResponse struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// Report — ответ /health/ready и /health/deep.
type Report struct {
	Status    string                    `json:"status"`
	Checks    map[string]ComponentCheck `json:"checks"`
	Timestamp time.Time                 `json:"timestamp"`
}

// ComponentCheck — проверка одной зависимости.
type ComponentCheck struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latencyMs"`
	Error     string `json:"error,omitempty"`
}
