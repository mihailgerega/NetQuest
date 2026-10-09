package engine

import "strings"

// Коды ошибок симуляции. Код попадает в summary.metadata.errorCode и в
// протокольный разбор; по нему фронтенд выбирает текст и совет пользователю.
const (
	codeSourceNodeRequired     = "SOURCE_NODE_REQUIRED"
	codeSourceNodeNotFound     = "SOURCE_NODE_NOT_FOUND"
	codeSourceNodeMustBeClient = "SOURCE_NODE_MUST_BE_CLIENT"
	codeSourceNodeDown         = "SOURCE_NODE_DOWN"
	codeDestinationNotFound    = "DESTINATION_NOT_FOUND"
	codeRouteNotFound          = "ROUTE_NOT_FOUND"
	codeDNSRecordNotFound      = "DNS_RECORD_NOT_FOUND"
	codeFirewallDenied         = "FIREWALL_DENIED"
	codeServerPortClosed       = "SERVER_PORT_CLOSED"
	codeNoHealthyBackends      = "NO_HEALTHY_BACKENDS"
	codeTopologyInvalid        = "TOPOLOGY_INVALID"
	codeSimulationFailed       = "SIMULATION_FAILED"
)

// errorCodeForMessage определяет код ошибки по тексту провала.
//
// Код выводится из текста, а не передаётся в fail отдельно: так тексты
// провалов остаются единственным источником правды, а неизвестный текст
// получает общий код SIMULATION_FAILED. Порядок веток важен — первая
// подходящая побеждает.
func errorCodeForMessage(message string) string {
	switch {
	case message == "sourceNodeId is required":
		return codeSourceNodeRequired
	case message == "source node does not exist":
		return codeSourceNodeNotFound
	case message == "source node must be a client":
		return codeSourceNodeMustBeClient
	case message == "source client is down":
		return codeSourceNodeDown
	case strings.HasPrefix(message, "ping destination not found"):
		return codeDestinationNotFound
	case strings.HasPrefix(message, "no route from"):
		return codeRouteNotFound
	case strings.HasPrefix(message, "DNS NXDOMAIN"):
		return codeDNSRecordNotFound
	case strings.Contains(message, "Firewall"):
		return codeFirewallDenied
	case strings.HasPrefix(message, "server does not listen on"):
		return codeServerPortClosed
	case message == errNoHealthyBackends:
		return codeNoHealthyBackends
	case strings.Contains(message, "topology"):
		return codeTopologyInvalid
	default:
		return codeSimulationFailed
	}
}

// errorUserMessage — понятное пользователю описание ошибки по коду.
func errorUserMessage(code string) string {
	switch code {
	case codeSourceNodeRequired:
		return "Выберите Client, от которого нужно отправить request."
	case codeSourceNodeNotFound:
		return "Source node не найден."
	case codeSourceNodeMustBeClient:
		return "Source node должен быть Client."
	case codeSourceNodeDown:
		return "Выбранный Client недоступен."
	case codeDestinationNotFound:
		return "Назначение не найдено."
	case codeRouteNotFound:
		return "Маршрут не найден."
	case codeDNSRecordNotFound:
		return "DNS record отсутствует."
	case codeFirewallDenied:
		return "Firewall заблокировал packet."
	case codeServerPortClosed:
		return "Server does not listen on the requested port."
	case codeNoHealthyBackends:
		return "Нет доступных healthy backend."
	default:
		return "Simulation завершилась ошибкой."
	}
}

// errorSuggestedFix — совет, что исправить в топологии, по коду ошибки.
func errorSuggestedFix(code string) string {
	switch code {
	case codeServerPortClosed:
		return "Open the requested server port or change the request/backend service port."
	case codeSourceNodeRequired, codeSourceNodeNotFound, codeSourceNodeMustBeClient, codeSourceNodeDown:
		return "Проверьте выбранный source Client и его status."
	case codeDestinationNotFound:
		return "Проверьте target node, hostname или IP."
	case codeRouteNotFound:
		return "Проверьте links, route table, gateway и status nodes."
	case codeDNSRecordNotFound:
		return "Добавьте корректный DNS A record."
	case codeFirewallDenied:
		return "Добавьте allow rule для нужного protocol/port выше deny rule."
	case codeNoHealthyBackends:
		return "Добавьте healthy reachable Server в Load Balancer backend pool."
	default:
		return "Откройте Timeline и Validation Advisor, чтобы найти проблемный stage."
	}
}
