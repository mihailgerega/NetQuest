package validator

import (
	"fmt"
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// nilString — так fmt.Sprint печатает nil. Значение ключа, которого нет
// в JSON-объекте (или null), после fmt.Sprint выглядит именно так.
const nilString = "<nil>"

// checkLoadBalancerBackends проверяет пул серверов Load Balancer:
// config.backends — массив объектов {"nodeId": ...}, где каждый nodeId
// уникален и ссылается на существующий узел типа server.
//
// Пула нет совсем — это не ошибка: Load Balancer может находить серверы сам
// (autoDiscoverConnectedServers), а пустой пул разбирает советник.
func checkLoadBalancerBackends(node model.Node, nodeIndex int, nodesByID map[string]model.Node, rep *report) {
	raw, exists := node.Config["backends"]
	if !exists || raw == nil {
		return
	}

	backends, ok := raw.([]any)
	if !ok {
		rep.add(fmt.Sprintf("nodes[%d].config.backends", nodeIndex), "load balancer backends must be an array")
		return
	}

	seen := map[string]struct{}{}

	for i, item := range backends {
		path := fmt.Sprintf("nodes[%d].config.backends[%d]", nodeIndex, i)

		backend, ok := item.(map[string]any)
		if !ok {
			rep.add(path, "backend entry must be an object")
			continue
		}

		// fmt.Sprint, а не приведение к string: nodeId может прийти числом.
		nodeID := strings.TrimSpace(fmt.Sprint(backend["nodeId"]))
		if nodeID == "" || nodeID == nilString {
			rep.add(path+".nodeId", "backend nodeId is required")
			continue
		}

		if _, exists := seen[nodeID]; exists {
			rep.add(path+".nodeId", "backend nodeId must be unique")
			continue
		}

		seen[nodeID] = struct{}{}

		backendNode, exists := nodesByID[nodeID]
		if !exists {
			rep.add(path+".nodeId", "backend node does not exist")
			continue
		}

		if backendNode.Type != model.NodeTypeServer {
			rep.add(path+".nodeId", "backend node must reference a server")
		}
	}
}
