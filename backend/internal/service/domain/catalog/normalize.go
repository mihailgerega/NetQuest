package catalog

import (
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// normalizeLegacyHints нормализует обычные подсказки и выбрасывает пустые.
func normalizeLegacyHints(hints []string) []string {
	normalized := make([]string, 0, len(hints))
	for _, hint := range hints {
		if hint == "" {
			continue
		}
		normalized = append(normalized, normalizeHintText(hint))
	}
	return normalized
}

// normalizeTextList нормализует список текстов, сохраняя пустые элементы.
func normalizeTextList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, normalizeHintText(value))
	}
	return normalized
}

// normalizeCheckSpecs нормализует тексты проверок: заголовок, сообщение и подсказку.
func normalizeCheckSpecs(checks []model.CheckSpec) []model.CheckSpec {
	normalized := make([]model.CheckSpec, 0, len(checks))
	for _, check := range checks {
		check.Title = normalizeHintText(check.Title)
		check.Message = normalizeHintText(check.Message)
		check.Hint = normalizeHintText(check.Hint)
		normalized = append(normalized, check)
	}
	return normalized
}

// normalizeHintActions нормализует действия подсказки; пустой список остаётся как был.
func normalizeHintActions(actions []string) []string {
	if len(actions) == 0 {
		return actions
	}
	normalized := make([]string, 0, len(actions))
	for _, action := range actions {
		normalized = append(normalized, normalizeHintText(action))
	}
	return normalized
}

// hintReplacer переводит англоязычные термины из текстов квестов в единую
// русскую терминологию интерфейса.
//
// strings.Replacer идёт по тексту слева направо и в каждой позиции применяет
// первую по списку пару, которая там совпала. Поэтому фраза стоит выше слова,
// с которого она начинается: "backend pool" раньше "backend",
// "Packet Inspector" раньше "Packet". Порядок пар — часть результата.
//
// Replacer неизменяем и безопасен для параллельного использования, поэтому
// создаётся один раз на пакет, а не на каждый вызов.
var hintReplacer = strings.NewReplacer(
	"DNS node", "DNS-узел",
	"DNS record", "DNS-запись",
	"DNS resolver", "DNS-сервер",
	"Server nodes", "серверы",
	"Server node", "сервер",
	"node/link", "узел или канал связи",
	"node status", "состояние узла",
	"link status", "состояние канала связи",
	"status link", "состояние канала связи",
	"links", "каналы связи",
	"link", "канал связи",
	"Inspector", "инспектор",
	"Packet Inspector", "инспектор пакета",
	"Protocol Inspector", "протокольный разбор",
	"A record", "A-запись",
	"Record", "Запись",
	"Value", "Значение",
	"backend pool", "пул серверов",
	"Backend pool", "Пул серверов",
	"fallback backend", "резервный сервер",
	"skipped backends", "пропущенные серверы",
	"LB decision", "решение Load Balancer",
	"Backend", "Сервер приложения",
	"backend", "сервер приложения",
	"healthy backend", "исправный сервер",
	"healthy", "исправный",
	"reachable", "достижимый",
	"Down backend", "Выключенный сервер",
	"down", "выключен",
	"skippedBackends", "список пропущенных серверов",
	"stale backend", "устаревшая ссылка на сервер",
	"silently ignore", "молча игнорировать",
	"source node", "исходный узел",
	"source", "источник",
	"destination", "назначение",
	"packet loss", "потеря пакетов",
	"Packet", "Пакет",
	"packet", "пакет",
	"Simulation", "Симуляция",
	"simulation", "симуляция",
	"error", "ошибка",
	"failed event", "событие с ошибкой",
	"failed check", "проверку с ошибкой",
	"config", "настройки",
	"canvas", "рабочем поле",
	"latencyBreakdown", "разбор задержки",
	"Latency", "Задержка",
	"latency", "задержку",
	"Graph path", "Алгоритм пути по графу",
	"lowest-latency", "с наименьшей задержкой",
	"Fast-link", "Быстрый канал",
	"fast-link", "быстрый канал",
	"slow router", "медленный маршрутизатор",
	"Primary route", "Основной маршрут",
	"backup path", "резервный путь",
	"active path", "активный путь",
	"Direct Server-1 access", "Прямой доступ к Server-1",
	"Direct access", "Прямой доступ",
	"direct IP", "прямой IP-адрес",
	"Path", "Путь",
	"path", "путь",
	"route cost", "стоимость маршрута",
	"routing table", "таблицу маршрутизации",
	"routed path", "маршрут",
	"Default gateway", "Шлюз по умолчанию",
	"gateway", "шлюз",
	"Default route", "Маршрут по умолчанию",
	"defaultGateway", "шлюз по умолчанию",
	"Request", "Запрос",
	"request", "запрос",
	"TLS hostname", "имя TLS-сертификата",
	"hostname", "имя хоста",
	"backend-checker'ом", "серверной проверкой",
	"port", "порт",
	"protocol", "протокол",
)

// normalizeHintText приводит текст к единой терминологии (см. hintReplacer).
func normalizeHintText(value string) string {
	return hintReplacer.Replace(value)
}

// maxInt — большее из двух чисел.
func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

// containsFold — содержит ли value подстроку needle без учёта регистра ASCII.
// Категории квестов сравниваются по латинским названиям ("DNS", "Load Balancer"),
// поэтому полноценный Unicode case folding не нужен.
func containsFold(value, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(value) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(value); i++ {
		if equalFoldASCII(value[i:i+len(needle)], needle) {
			return true
		}
	}
	return false
}

// equalFoldASCII сравнивает строки одинаковой длины без учёта регистра ASCII.
func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca := a[i]
		cb := b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
