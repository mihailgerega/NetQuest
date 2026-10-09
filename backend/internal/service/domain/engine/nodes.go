package engine

import (
	"net/url"
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// Статусы, при которых узел или канал не пропускает трафик.
const (
	statusDown     = "down"
	statusIsolated = "isolated"
)

// findNode ищет цель ping: по ID узла, затем по IP, config.hostname или имени.
func (r *runner) findNode(target string) (model.Node, bool) {
	if node, ok := r.nodes[target]; ok {
		return node, true
	}

	for _, node := range r.doc.Nodes {
		if nodeIP(node) == target || stringValue(node.Config["hostname"]) == target || node.Name == target {
			return node, true
		}
	}

	return model.Node{}, false
}

// findNodeByIP ищет узел, чей основной IP равен ip (первый в порядке документа).
func (r *runner) findNodeByIP(ip string) (model.Node, bool) {
	for _, node := range r.doc.Nodes {
		if nodeIP(node) == ip {
			return node, true
		}
	}

	return model.Node{}, false
}

// hasDownInfrastructure сообщает, есть ли в топологии выключенный узел или канал.
func (r *runner) hasDownInfrastructure() bool {
	for _, node := range r.doc.Nodes {
		if nodeDown(node) {
			return true
		}
	}

	for _, link := range r.doc.Links {
		if linkDown(link) {
			return true
		}
	}

	return false
}

// targetHost достаёт хост из цели сценария: "https://api.local/users" → "api.local".
// Цель без схемы (имя или IP) возвращается как есть, без пробелов по краям.
func targetHost(target string) string {
	parsed, err := url.Parse(target)
	if err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}

	return strings.TrimSpace(target)
}

// nodeName — подпись узла: имя, а без имени — ID.
func nodeName(node model.Node) string {
	if strings.TrimSpace(node.Name) != "" {
		return strings.TrimSpace(node.Name)
	}

	return node.ID
}

// nodeIP — основной IP узла: config.ip, а без него — IP первого интерфейса.
func nodeIP(node model.Node) string {
	if ip := stringValue(node.Config["ip"]); ip != "" {
		return ip
	}

	for _, iface := range anySlice(node.Config["interfaces"]) {
		if ip := stringValue(anyMap(iface)["ip"]); ip != "" {
			return ip
		}
	}

	return ""
}

// nodeDown — узел выключен или изолирован. Статус берётся из поля status,
// а если оно пустое — из config.status (так его хранят старые топологии).
func nodeDown(node model.Node) bool {
	status := node.Status
	if status == "" {
		status = stringValue(node.Config["status"])
	}

	return status == statusDown || status == statusIsolated
}

// linkDown — канал выключен. Статус — как у nodeDown: поле или config.status.
func linkDown(link model.Link) bool {
	status := link.Status
	if status == "" {
		status = stringValue(link.Config["status"])
	}

	return status == statusDown
}
