package engine

import (
	"math"

	"github.com/netquest/netquest/backend/internal/model"
)

// Два режима поиска маршрута:
//
//   - graph_path — кратчайший по задержке путь по активным каналам (Дейкстра).
//     Включается, когда в топологии нет никаких сетевых настроек;
//   - route_table — как в настоящей сети: шлюз по умолчанию у клиента
//     и таблицы маршрутизации у роутеров. Включается, как только у источника
//     или цели есть cidr/шлюз/интерфейсы/маршруты или у любого роутера есть routes.

// defaultLinkLatency — задержка канала, если в config нет ни latencyMs, ни cost.
const defaultLinkLatency = 10

// findPath ищет маршрут от sourceID до targetID и его суммарную задержку.
// Объяснение выбора (или причину неудачи) оставляет в r.routeInfo.
func (r *runner) findPath(sourceID, targetID string) ([]string, int64, bool) {
	r.routeInfo = ""

	if r.usesRoutingTables(sourceID, targetID) {
		path, latency, ok := r.findRoutedPath(sourceID, targetID)
		if ok {
			return path, latency, true
		}

		if r.routeInfo == "" {
			r.routeInfo = "Route table mode is enabled, but no matching route could be built."
		}

		return nil, 0, false
	}

	path, latency, ok := r.findGraphPath(sourceID, targetID)
	if ok {
		r.routeInfo = "Selected lowest-latency active graph path."
	}

	return path, latency, ok
}

// graphEdge — ребро графа каналов: куда ведёт и сколько стоит.
type graphEdge struct {
	to      string
	latency int64
}

// findGraphPath — алгоритм Дейкстры по активным каналам и исправным узлам.
// Вес ребра — задержка канала.
//
// Очередь с приоритетом не нужна: узлов не больше сотни, поэтому минимум
// ищется перебором map. Перебор map в Go идёт в случайном порядке — при равных
// расстояниях выбор между путями одинаковой стоимости не фиксирован.
func (r *runner) findGraphPath(sourceID, targetID string) ([]string, int64, bool) {
	if sourceID == "" || targetID == "" {
		return nil, 0, false
	}

	if sourceID == targetID {
		return []string{sourceID}, 0, true
	}

	if nodeDown(r.nodes[sourceID]) || nodeDown(r.nodes[targetID]) {
		return nil, 0, false
	}

	dist, prev := dijkstra(r.activeGraph(), sourceID, targetID)

	total, ok := dist[targetID]
	if !ok {
		return nil, 0, false
	}

	path, ok := restorePath(prev, sourceID, targetID)
	if !ok {
		return nil, 0, false
	}

	return path, total, true
}

// dijkstra считает кратчайшие расстояния от sourceID и предков на кратчайших
// путях. Останавливается, как только из очереди достаётся targetID: расстояние
// до него уже окончательное, остальные узлы не нужны.
func dijkstra(graph map[string][]graphEdge, sourceID, targetID string) (map[string]int64, map[string]string) {
	dist := map[string]int64{sourceID: 0}
	prev := map[string]string{}
	visited := map[string]bool{}

	for {
		current := closestUnvisited(dist, visited)

		// Достижимые узлы кончились или до цели уже дошли — расстояние до неё окончательное.
		if current == "" || current == targetID {
			return dist, prev
		}

		visited[current] = true

		for _, next := range graph[current] {
			candidate := dist[current] + next.latency
			if existing, ok := dist[next.to]; !ok || candidate < existing {
				dist[next.to] = candidate
				prev[next.to] = current
			}
		}
	}
}

// closestUnvisited — непосещённый узел с наименьшим расстоянием; "" — таких нет.
func closestUnvisited(dist map[string]int64, visited map[string]bool) string {
	current := ""
	best := int64(math.MaxInt64)

	for nodeID, value := range dist {
		if !visited[nodeID] && value < best {
			current = nodeID
			best = value
		}
	}

	return current
}

// activeGraph строит список смежности из активных каналов между исправными
// узлами. Каналы ненаправленные — ребро добавляется в обе стороны.
func (r *runner) activeGraph() map[string][]graphEdge {
	graph := map[string][]graphEdge{}

	for _, link := range r.doc.Links {
		if linkDown(link) || nodeDown(r.nodes[link.SourceNodeID]) || nodeDown(r.nodes[link.TargetNodeID]) {
			continue
		}

		latency := linkLatency(link)
		graph[link.SourceNodeID] = append(graph[link.SourceNodeID], graphEdge{to: link.TargetNodeID, latency: latency})
		graph[link.TargetNodeID] = append(graph[link.TargetNodeID], graphEdge{to: link.SourceNodeID, latency: latency})
	}

	return graph
}

// restorePath восстанавливает путь по цепочке предков: от цели назад к источнику,
// затем разворачивает.
func restorePath(prev map[string]string, sourceID, targetID string) ([]string, bool) {
	path := []string{targetID}

	for current := targetID; current != sourceID; {
		parent, ok := prev[current]
		if !ok {
			return nil, false
		}

		path = append(path, parent)
		current = parent
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path, true
}

// findDirectPath ищет прямой активный канал между двумя узлами — так в режиме
// таблиц маршрутизации проверяется, что следующий хоп действительно соседний.
func (r *runner) findDirectPath(sourceID, targetID string) ([]string, int64, bool) {
	if sourceID == "" || targetID == "" {
		return nil, 0, false
	}

	if sourceID == targetID {
		return []string{sourceID}, 0, true
	}

	if nodeDown(r.nodes[sourceID]) || nodeDown(r.nodes[targetID]) {
		return nil, 0, false
	}

	for _, link := range r.doc.Links {
		if linkDown(link) {
			continue
		}

		forward := link.SourceNodeID == sourceID && link.TargetNodeID == targetID
		backward := link.SourceNodeID == targetID && link.TargetNodeID == sourceID

		if forward || backward {
			return []string{sourceID, targetID}, linkLatency(link), true
		}
	}

	return nil, 0, false
}

// linkLatency — задержка канала: latencyMs, иначе cost, иначе defaultLinkLatency.
func linkLatency(link model.Link) int64 {
	return int64(intValue(link.Config["latencyMs"], intValue(link.Config["cost"], defaultLinkLatency)))
}

// usesRoutingTables решает, искать ли маршрут по таблицам маршрутизации
// (см. описание режимов в начале файла).
func (r *runner) usesRoutingTables(sourceID, targetID string) bool {
	source, sourceOK := r.nodes[sourceID]
	target, targetOK := r.nodes[targetID]

	if !sourceOK || !targetOK {
		return false
	}

	if hasRoutingConfig(source) || hasRoutingConfig(target) {
		return true
	}

	for _, node := range r.doc.Nodes {
		if node.Type == model.NodeTypeRouter && len(anySlice(node.Config["routes"])) > 0 {
			return true
		}
	}

	return false
}

// routingAlgorithm — название режима для события route.selected.
func (r *runner) routingAlgorithm(sourceID, targetID string) string {
	if r.usesRoutingTables(sourceID, targetID) {
		return "route_table"
	}

	return "graph_path"
}

// hasRoutingConfig сообщает, есть ли у узла сетевые настройки L3.
func hasRoutingConfig(node model.Node) bool {
	return stringValue(node.Config["cidr"]) != "" ||
		stringValue(node.Config["defaultGateway"]) != "" ||
		len(anySlice(node.Config["interfaces"])) > 0 ||
		len(anySlice(node.Config["routes"])) > 0
}
