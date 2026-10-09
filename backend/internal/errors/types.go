package errs

import "github.com/netquest/netquest/backend/internal/model"

// ValidationError — 422 validation_failed: запрос разобран, но данные не проходят
// правила сервиса (пустое имя, неизвестная видимость, топология с ошибками).
//
// Не sentinel, а тип: у каждой такой ошибки свой текст и свои подробности —
// например, список допустимых значений или результат валидации топологии.
// Details уходит клиенту в поле error.details как есть.
type ValidationError struct {
	Message string
	Details any
}

// Error возвращает текст для клиента — он же message в ответе.
func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError создаёт ошибку валидации. Возвращает error, а не
// *ValidationError: так вызывающий код не получит «nil-указатель в интерфейсе».
func NewValidationError(message string, details any) error {
	return &ValidationError{Message: message, Details: details}
}

// InvalidJSONError — 400 bad_request: тело запроса не JSON нужной формы
// (битый синтаксис, неизвестное поле, неверный тип значения).
type InvalidJSONError struct {
	Cause error
}

// Error возвращает текст для клиента: "invalid JSON body: <ошибка encoding/json>".
func (e *InvalidJSONError) Error() string {
	return "invalid JSON body: " + e.Cause.Error()
}

// Unwrap открывает исходную ошибку encoding/json для errors.Is / errors.As.
func (e *InvalidJSONError) Unwrap() error {
	return e.Cause
}

// RateLimitError — 429 rate_limited: клиент превысил лимит запросов в минуту.
// Limit уходит клиенту в details, чтобы он знал, сколько ему разрешено.
type RateLimitError struct {
	Limit int
}

// Error возвращает текст для клиента; сам лимит — в details ответа.
func (e *RateLimitError) Error() string {
	return "rate limit exceeded"
}

// TopologyInvalidError — движок симуляции отказался запускать сценарий:
// топология не прошла валидацию. Validation — полный список найденных ошибок.
//
// Тип значения, а не указателя: движок возвращает его как есть, а errors.As
// ищет в цепочке именно значение этого типа.
type TopologyInvalidError struct {
	Validation model.ValidationResult
}

// Error — фиксированный текст: он сохраняется в simulations.error_message
// и попадает в результат проверки квеста, поэтому подробностей в нём нет.
func (e TopologyInvalidError) Error() string {
	return "topology is invalid"
}
