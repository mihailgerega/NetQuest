package advisor

import (
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// categoryRouting — категория замечаний о маршрутизации.
const categoryRouting = "Routing"

// analyzeRouting проверяет связность и шлюзы:
//
//   - если сценарий задаёт источник и цель-узел, между ними должен быть путь
//     по активным каналам (цель-URL не проверяется: её узел знает только DNS);
//   - у клиента с подсетью (cidr) должен быть шлюз по умолчанию, иначе
//     в другие подсети ему не выйти. Это только информация: клиенту может
//     и не нужно уходить из своей подсети.
func analyzeRouting(doc model.Document, scenario *model.Scenario) []model.Issue {
	issues := []model.Issue{}

	if scenario != nil && scenario.SourceNodeID != "" && scenario.Target != "" {
		targetID := scenario.Target
		if strings.HasPrefix(targetID, "http") {
			targetID = ""
		}

		if targetID != "" && !graphReachable(doc, scenario.SourceNodeID, targetID) {
			issues = append(issues, model.Issue{
				Severity:       model.IssueSeverityError,
				Category:       categoryRouting,
				Code:           "ROUTE_MISSING",
				Title:          "Маршрут не найден",
				Message:        "Между source и target нет active path.",
				AffectedNodeID: scenario.SourceNodeID,
				SuggestedFix:   "Проверьте links, status nodes и routing table.",
			})
		}
	}

	for _, node := range doc.Nodes {
		if node.Type != model.NodeTypeClient {
			continue
		}

		cidr := stringValue(node.Config["cidr"])
		if cidr != "" && stringValue(node.Config["defaultGateway"]) == "" {
			issues = append(issues, model.Issue{
				Severity:       model.IssueSeverityInfo,
				Category:       categoryRouting,
				Code:           "DEFAULT_GATEWAY_MISSING",
				Title:          "Default gateway не задан",
				Message:        node.ID + " имеет CIDR, но не имеет defaultGateway.",
				AffectedNodeID: node.ID,
				SuggestedFix:   "Укажите defaultGateway для доступа в другие subnet.",
			})
		}
	}

	return issues
}

// graphReachable — обход в ширину по активным каналам. Выключенные узлы
// (status "down") не проходимы, но сам источник проверяется без учёта статуса:
// его состояние разбирает analyzeSource.
//
// В отличие от движка, здесь нет задержек, таблиц маршрутизации и config.status —
// только «есть ли вообще путь».
func graphReachable(doc model.Document, sourceID, targetID string) bool {
	if sourceID == targetID {
		return true
	}

	nodes := map[string]model.Node{}
	for _, node := range doc.Nodes {
		nodes[node.ID] = node
	}

	queue := []string{sourceID}
	visited := map[string]bool{sourceID: true}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, link := range doc.Links {
			if link.Status == "down" {
				continue
			}

			next := neighbour(link, current)
			if next == "" || visited[next] || nodes[next].Status == "down" {
				continue
			}

			if next == targetID {
				return true
			}

			visited[next] = true
			queue = append(queue, next)
		}
	}

	return false
}

// neighbour — другой конец канала, если канал касается узла; иначе "".
func neighbour(link model.Link, nodeID string) string {
	switch nodeID {
	case link.SourceNodeID:
		return link.TargetNodeID
	case link.TargetNodeID:
		return link.SourceNodeID
	default:
		return ""
	}
}
