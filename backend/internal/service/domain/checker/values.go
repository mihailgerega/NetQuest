package checker

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

// Хелперы чтения config — как в движке, но с отличиями, на которые опираются
// проверки квестов: defaultString принимает строку, а intValue разбирает строку
// через fmt.Sscanf ("443/tcp" → 443), а не strconv.Atoi. Поэтому хелперы свои,
// а не общие с движком.

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

// defaultString — строка без пробелов по краям или fallback, если она пустая.
func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return strings.TrimSpace(value)
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

// cidrOrIPMatches — подходит ли адрес под назначение правила firewall.
// "", "any" и 0.0.0.0/0 подходят ко всему; иначе — точное совпадение строк
// или попадание IP в CIDR.
func cidrOrIPMatches(cidr, ipValue string) bool {
	if cidr == "" || cidr == "any" || cidr == "0.0.0.0/0" {
		return true
	}

	if cidr == ipValue {
		return true
	}

	ip := net.ParseIP(ipValue)
	if ip == nil || !strings.Contains(cidr, "/") {
		return false
	}

	_, network, err := net.ParseCIDR(cidr)

	return err == nil && network.Contains(ip)
}
