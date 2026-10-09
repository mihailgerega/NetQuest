package validator

import (
	"fmt"
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// checkNodes проверяет ID и типы узлов и возвращает узлы по ID — они нужны
// проверкам каналов и настроек.
//
// Узел без ID дальше не проверяется (тип у него не смотрим): ссылаться на него
// всё равно нельзя. При повторе ID в карте остаётся последний узел.
func checkNodes(doc model.Document, allowedTypes map[model.NodeType]struct{}, rep *report) map[string]model.Node {
	nodesByID := make(map[string]model.Node, len(doc.Nodes))

	for i, node := range doc.Nodes {
		path := fmt.Sprintf("nodes[%d]", i)

		nodeID := strings.TrimSpace(node.ID)
		if nodeID == "" {
			rep.add(path+".id", "node id is required")
			continue
		}

		if _, exists := nodesByID[nodeID]; exists {
			rep.add(path+".id", "node id must be unique")
		}

		nodesByID[nodeID] = node

		if _, ok := allowedTypes[node.Type]; !ok {
			rep.addf(path+".type", "unsupported node type %q", node.Type)
		}
	}

	return nodesByID
}

// checkLinks проверяет, что у каждого канала есть ID и оба конца ссылаются
// на существующие узлы.
func checkLinks(doc model.Document, nodesByID map[string]model.Node, rep *report) {
	for i, link := range doc.Links {
		path := fmt.Sprintf("links[%d]", i)

		if strings.TrimSpace(link.ID) == "" {
			rep.add(path+".id", "link id is required")
		}

		checkLinkEnd(strings.TrimSpace(link.SourceNodeID), path+".sourceNodeId", "source", nodesByID, rep)
		checkLinkEnd(strings.TrimSpace(link.TargetNodeID), path+".targetNodeId", "target", nodesByID, rep)
	}
}

// checkLinkEnd проверяет один конец канала; side — "source" или "target"
// для текста сообщения.
func checkLinkEnd(nodeID, path, side string, nodesByID map[string]model.Node, rep *report) {
	if nodeID == "" {
		rep.add(path, side+" node is required")
		return
	}

	if _, ok := nodesByID[nodeID]; !ok {
		rep.add(path, side+" node does not exist")
	}
}

// checkNodeConfigs проверяет настройки, у которых есть правила:
// пул серверов Load Balancer и открытые порты Server.
func checkNodeConfigs(doc model.Document, nodesByID map[string]model.Node, rep *report) {
	for i, node := range doc.Nodes {
		switch node.Type {
		case model.NodeTypeLoadBalancer:
			checkLoadBalancerBackends(node, i, nodesByID, rep)
		case model.NodeTypeServer:
			checkServerOpenPorts(node, i, rep)
		}
	}
}
