package advisor

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Хелперы чтения config узлов и каналов. intValue разбирает строки через
// fmt.Sscanf — как чекер квестов, чтобы правило HIGH_LATENCY_LINK советника
// и проверка advisor_check квеста одинаково читали latencyMs.

// anyMap возвращает значение как JSON-объект; всё остальное — пустой объект.
func anyMap(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}

	return map[string]any{}
}

// anySlice возвращает значение как JSON-массив; всё остальное — nil.
func anySlice(value any) []any {
	if items, ok := value.([]any); ok {
		return items
	}

	return nil
}

// stringValue печатает значение строкой без пробелов по краям; nil — пустая строка.
func stringValue(value any) string {
	if value == nil {
		return ""
	}

	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}

	return strings.TrimSpace(fmt.Sprint(value))
}

// defaultString — stringValue, но пустое значение заменяется fallback.
func defaultString(value any, fallback string) string {
	if s := stringValue(value); s != "" {
		return s
	}

	return fallback
}

// intValue читает целое; строка разбирается по ведущему числу.
func intValue(value any, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	case string:
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}

	return fallback
}
