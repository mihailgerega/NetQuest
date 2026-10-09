package engine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Хелперы чтения config узлов и каналов. Config — map[string]any после
// encoding/json: числа там float64, массивы — []any, объекты — map[string]any.
// Пользователь мог ввести значение строкой ("5" вместо 5), поэтому хелперы
// понимают и строки. Неподходящее значение — fallback, а не ошибка: движок
// объясняет последствия кривой настройки событиями, а не падением.

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

// intValue читает целое: дробная часть float64 отбрасывается, строка
// разбирается strconv.Atoi.
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
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}

	return fallback
}

// floatValue читает число с плавающей точкой (проценты потерь).
func floatValue(value any, fallback float64) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}

	return fallback
}

// boolValue читает флаг: bool или строку, понятную strconv.ParseBool.
func boolValue(value any, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}

	return fallback
}
