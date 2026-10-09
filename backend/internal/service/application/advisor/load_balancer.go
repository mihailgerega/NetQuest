package advisor

import (
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// categoryLoadBalancer — категория замечаний о Load Balancer.
const categoryLoadBalancer = "Load Balancer"

// analyzeLoadBalancers проверяет пул каждого Load Balancer'а: он не пуст,
// а каждая запись ссылается на существующий узел типа server.
//
// Пересекается с валидатором намеренно: валидатор отвечает «можно ли сохранить»,
// а советник объясняет, что именно сломается и как это исправить.
func analyzeLoadBalancers(doc model.Document) []model.Issue {
	issues := []model.Issue{}

	nodes := map[string]model.Node{}
	for _, node := range doc.Nodes {
		nodes[node.ID] = node
	}

	for _, node := range doc.Nodes {
		if node.Type != model.NodeTypeLoadBalancer {
			continue
		}

		backends := anySlice(node.Config["backends"])
		if len(backends) == 0 {
			issues = append(issues, model.Issue{
				Severity:       model.IssueSeverityError,
				Category:       categoryLoadBalancer,
				Code:           "LB_BACKEND_POOL_EMPTY",
				Title:          "Load Balancer backend pool пустой",
				Message:        "Load Balancer не сможет выбрать backend для HTTPS request.",
				AffectedNodeID: node.ID,
				SuggestedFix:   "Добавьте хотя бы один healthy Server в backend pool.",
			})

			continue
		}

		for _, item := range backends {
			if issue, ok := backendIssue(node, nodes, stringValue(anyMap(item)["nodeId"])); ok {
				issues = append(issues, issue)
			}
		}
	}

	return issues
}

// backendIssue — замечание к одной записи пула: узла нет или он не server.
func backendIssue(lb model.Node, nodes map[string]model.Node, backendID string) (model.Issue, bool) {
	backend, ok := nodes[backendID]
	if !ok {
		return model.Issue{
			Severity:       model.IssueSeverityError,
			Category:       categoryLoadBalancer,
			Code:           "LB_STALE_BACKEND",
			Title:          "Backend ссылается на удалённый node",
			Message:        fmt.Sprintf("Load Balancer содержит backend %s, но такого node больше нет.", backendID),
			AffectedNodeID: lb.ID,
			SuggestedFix:   "Удалите stale backend из pool или добавьте node обратно.",
		}, true
	}

	if backend.Type != model.NodeTypeServer {
		return model.Issue{
			Severity:       model.IssueSeverityError,
			Category:       categoryLoadBalancer,
			Code:           "LB_BACKEND_NOT_SERVER",
			Title:          "Backend не является Server",
			Message:        backendID + " не является Server node.",
			AffectedNodeID: lb.ID,
			SuggestedFix:   "Оставьте в backend pool только Server nodes.",
		}, true
	}

	return model.Issue{}, false
}
