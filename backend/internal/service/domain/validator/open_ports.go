package validator

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// Диапазон номеров TCP/UDP-портов.
const (
	minPort = 1
	maxPort = 65535
)

// checkServerOpenPorts проверяет config.openPorts у Server. Элемент — либо
// просто номер порта (443 — это tcp/443), либо объект
// {"port": 443, "protocol": "tcp", "status": "open"}.
//
// Пара протокол/порт должна быть уникальной: два tcp/443 на одном сервере
// сделали бы непонятным, открыт порт или закрыт.
func checkServerOpenPorts(node model.Node, nodeIndex int, rep *report) {
	raw, exists := node.Config["openPorts"]
	if !exists || raw == nil {
		return
	}

	ports, ok := raw.([]any)
	if !ok {
		rep.add(fmt.Sprintf("nodes[%d].config.openPorts", nodeIndex), "server openPorts must be an array")
		return
	}

	seen := map[string]struct{}{}

	for i, item := range ports {
		path := fmt.Sprintf("nodes[%d].config.openPorts[%d]", nodeIndex, i)

		key, ok := checkOpenPortEntry(item, path, rep)
		if !ok {
			continue
		}

		if _, exists := seen[key]; exists {
			rep.add(path, "server open port must be unique")
		}

		seen[key] = struct{}{}
	}
}

// checkOpenPortEntry проверяет один элемент openPorts и возвращает его ключ
// "протокол/порт" для проверки уникальности. ok=false — элемент настолько
// сломан, что ключа у него нет.
func checkOpenPortEntry(item any, path string, rep *report) (string, bool) {
	if port, ok := numericPort(item); ok {
		if port < minPort || port > maxPort {
			rep.add(path, "server port must be between 1 and 65535")
		}

		return fmt.Sprintf("tcp/%d", port), true
	}

	openPort, ok := item.(map[string]any)
	if !ok {
		rep.add(path, "server openPort entry must be an object or port number")
		return "", false
	}

	port, ok := numericPort(openPort["port"])
	if !ok {
		rep.add(path+".port", "server openPort port is required")
		return "", false
	}

	if port < minPort || port > maxPort {
		rep.add(path+".port", "server port must be between 1 and 65535")
	}

	protocol := normalizedField(openPort["protocol"])
	if protocol == "" {
		protocol = "tcp"
	}

	if protocol != "tcp" && protocol != "udp" {
		rep.add(path+".protocol", "server openPort protocol must be tcp or udp")
	}

	switch normalizedField(openPort["status"]) {
	case "", "open", "closed", "filtered":
	default:
		rep.add(path+".status", "server openPort status must be open, closed or filtered")
	}

	return fmt.Sprintf("%s/%d", protocol, port), true
}

// normalizedField приводит значение поля к нижнему регистру без пробелов.
// Отсутствующее поле (nil) даёт пустую строку.
func normalizedField(value any) string {
	text := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
	if text == nilString {
		return ""
	}

	return text
}

// numericPort достаёт номер порта из значения JSON. Номер может прийти числом
// (encoding/json даёт float64), строкой ("443") или json.Number.
// Дробное число (443.5) портом не считается.
func numericPort(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		port := int(v)
		return port, float64(port) == v
	case int:
		return v, true
	case json.Number:
		i, err := v.Int64()
		return int(i), err == nil
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(v))
		return i, err == nil
	default:
		return 0, false
	}
}
