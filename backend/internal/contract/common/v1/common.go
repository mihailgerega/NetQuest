// Package commonv1 — общие типы HTTP-контракта NetQuest API: ответ-подтверждение
// и конверт ошибки.
//
// Пакеты internal/contract/*/v1 описывают JSON запросов и ответов API —
// ту же роль в solution играет сгенерированный ogen пакет orderv1. Здесь
// контракт написан вручную, а json-теги в нём — то, что видит фронтенд.
package commonv1

// OKResponse — ответ без данных: {"ok": true}.
type OKResponse struct {
	OK bool `json:"ok"`
}

// ErrorResponse — конверт любой ошибки API: {"error": {...}}.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody — описание ошибки.
//
// Code — машиночитаемый код (not_found, validation_failed...), Details —
// подробности (например, результат валидации) или null, Message — текст
// для пользователя, RequestID — ID запроса для поиска в логах.
//
// Поля в алфавитном порядке — в нём их сериализовал прежний map[string]any,
// и ответ остаётся побайтно прежним.
type ErrorBody struct {
	Code      string `json:"code"`
	Details   any    `json:"details"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}
