package config

import (
	"strconv"
	"strings"
	"time"
)

// Хелперы чтения переменных. Общее правило: переменная не задана, пустая или
// не разбирается — берём fallback. Падать на кривом значении не стали, чтобы
// опечатка в необязательной настройке не роняла сервис на старте.

// getString возвращает значение переменной как есть (без обрезки пробелов).
func getString(lookup LookupFunc, key, fallback string) string {
	if value, ok := lookup(key); ok && strings.TrimSpace(value) != "" {
		return value
	}

	return fallback
}

// getCSV разбирает список через запятую: "a, b,,c" → ["a", "b", "c"].
// Пустые элементы выбрасываются; если не осталось ни одного — fallback.
func getCSV(lookup LookupFunc, key string, fallback []string) []string {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}

	if len(out) == 0 {
		return fallback
	}

	return out
}

// getBool понимает всё, что понимает strconv.ParseBool: 1/0, true/false, T/F.
func getBool(lookup LookupFunc, key string, fallback bool) bool {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

// getInt читает целое число в десятичной записи.
func getInt(lookup LookupFunc, key string, fallback int) int {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

// getInt64 — как getInt, но для размеров в байтах, которым мало int32.
func getInt64(lookup LookupFunc, key string, fallback int64) int64 {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

// getDuration читает длительность в формате time.ParseDuration: "5s", "15m", "720h".
func getDuration(lookup LookupFunc, key string, fallback time.Duration) time.Duration {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
