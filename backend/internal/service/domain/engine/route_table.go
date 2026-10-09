package engine

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// Значения по умолчанию для маршрутов в config.routes.
const (
	defaultRouteDestination = "0.0.0.0/0" // маршрут без destination — маршрут по умолчанию
	defaultRouteMetric      = 100
	hostPrefixLen           = 32 // адрес без маски — это /32
)

// routeEntry — маршрут из таблицы роутера, подходящий для адреса назначения.
type routeEntry struct {
	destination string // CIDR назначения
	gateway     string // IP следующего хопа; пусто — назначение подключено напрямую
	metric      int
	prefixLen   int
}

// findRoutedPath строит маршрут так, как его прошёл бы настоящий IP-пакет:
//
//  1. цель в подсети источника — идём напрямую по графу;
//  2. иначе клиент отдаёт пакет своему шлюзу по умолчанию (defaultGateway);
//  3. дальше каждый роутер выбирает маршрут из своей таблицы
//     (самый длинный префикс, при равенстве — меньшая метрика) и передаёт
//     пакет следующему хопу, пока пакет не дойдёт до цели или до роутера,
//     к которому подсеть цели подключена напрямую.
//
// Причину неудачи оставляет в r.routeInfo — её увидит пользователь.
func (r *runner) findRoutedPath(sourceID, targetID string) ([]string, int64, bool) {
	source, sourceOK := r.nodes[sourceID]
	target, targetOK := r.nodes[targetID]

	if !sourceOK || !targetOK {
		r.routeInfo = "source or target node does not exist"
		return nil, 0, false
	}

	if nodeDown(source) || nodeDown(target) {
		r.routeInfo = "source or target node is down"
		return nil, 0, false
	}

	destinationIP := nodeIP(target)
	if destinationIP == "" {
		r.routeInfo = "target node has no IP address"
		return nil, 0, false
	}

	if sameSubnet(source, destinationIP) {
		path, latency, ok := r.findGraphPath(sourceID, targetID)
		if ok {
			r.routeInfo = "Destination is in the source subnet; direct route selected."
		}

		return path, latency, ok
	}

	walk := routeWalk{
		target:        target,
		destinationIP: destinationIP,
		path:          []string{sourceID},
		current:       source,
		visited:       map[string]bool{sourceID: true},
	}

	if source.Type != model.NodeTypeRouter && !r.stepToDefaultGateway(&walk) {
		return nil, 0, false
	}

	return r.walkRouteTables(&walk)
}

// routeWalk — состояние пошагового прохода по таблицам маршрутизации.
type routeWalk struct {
	target        model.Node
	destinationIP string
	path          []string
	latency       int64
	current       model.Node      // узел, на котором сейчас пакет
	visited       map[string]bool // защита от петель в таблицах
}

// stepToDefaultGateway — первый хоп клиента: к шлюзу по умолчанию.
// Шлюз должен быть соседом клиента по активному каналу.
func (r *runner) stepToDefaultGateway(walk *routeWalk) bool {
	gatewayIP := stringValue(walk.current.Config["defaultGateway"])
	if gatewayIP == "" {
		r.routeInfo = "source client has no defaultGateway for off-subnet destination"
		return false
	}

	gateway, ok := r.findNodeByAnyIP(gatewayIP)
	if !ok {
		r.routeInfo = "defaultGateway " + gatewayIP + " does not match any node interface"
		return false
	}

	path, latency, ok := r.findDirectPath(walk.current.ID, gateway.ID)
	if !ok {
		r.routeInfo = "defaultGateway is not reachable through active links"
		return false
	}

	walk.path = appendPath(walk.path, path)
	walk.latency += latency
	walk.current = gateway

	return true
}

// walkRouteTables передаёт пакет от роутера к роутеру по их таблицам.
// Хопов не больше, чем узлов (+2 на источник и шлюз): дальше — петля.
func (r *runner) walkRouteTables(walk *routeWalk) ([]string, int64, bool) {
	for range len(r.doc.Nodes) + 2 {
		if walk.current.ID == walk.target.ID {
			r.routeInfo = "Route table reached destination."
			return walk.path, walk.latency, true
		}

		if sameSubnet(walk.current, walk.destinationIP) {
			path, latency, ok := r.findGraphPath(walk.current.ID, walk.target.ID)
			if ok {
				walk.path = appendPath(walk.path, path)
				walk.latency += latency
				r.routeInfo = "Router has directly connected subnet for destination."

				return walk.path, walk.latency, true
			}
		}

		routes := routeCandidates(walk.current, walk.destinationIP)
		if len(routes) == 0 {
			r.routeInfo = "router " + walk.current.ID + " has no route matching " + walk.destinationIP
			return nil, 0, false
		}

		if !r.followBestRoute(walk, routes) {
			return nil, 0, false
		}
	}

	r.routeInfo = "route lookup exceeded maximum hop count"

	return nil, 0, false
}

// followBestRoute переходит к следующему хопу по первому подходящему маршруту.
//
// Маршруты уже отсортированы по приоритету. Маршрут пропускается, если его
// шлюз не найден, ведёт в уже пройденный узел (петля) или до шлюза нет
// активного канала — тогда пробуется следующий, как резервный маршрут
// в настоящей таблице. Если не подошёл ни один, в r.routeInfo остаётся
// причина отказа последнего.
func (r *runner) followBestRoute(walk *routeWalk, routes []routeEntry) bool {
	lastReason := ""

	for _, route := range routes {
		next := walk.target

		if route.gateway != "" {
			gatewayNode, found := r.findNodeByAnyIP(route.gateway)
			if !found {
				lastReason = "route gateway " + route.gateway + " does not match any node interface"
				continue
			}

			next = gatewayNode
		}

		if walk.visited[next.ID] {
			lastReason = "route table loop detected at " + next.ID
			continue
		}

		path, latency, ok := r.findDirectPath(walk.current.ID, next.ID)
		if !ok {
			lastReason = "next hop " + next.ID + " is not reachable through active links"
			continue
		}

		walk.path = appendPath(walk.path, path)
		walk.latency += latency
		walk.visited[next.ID] = true
		walk.current = next
		r.routeInfo = fmt.Sprintf("Route table selected %s via %s metric %d.", route.destination, defaultString(route.gateway, "direct"), route.metric)

		return true
	}

	if lastReason == "" {
		lastReason = "no reachable next hop for matching route"
	}

	r.routeInfo = lastReason

	return false
}

// routeCandidates возвращает маршруты узла, которые покрывают destinationIP,
// в порядке приоритета: длиннее префикс — точнее маршрут; при равной длине
// побеждает меньшая метрика. Сортировка стабильная: при полном равенстве
// сохраняется порядок из таблицы.
func routeCandidates(node model.Node, destinationIP string) []routeEntry {
	candidates := []routeEntry{}

	for _, item := range anySlice(node.Config["routes"]) {
		m := anyMap(item)

		destination := defaultString(m["destination"], defaultRouteDestination)
		if !cidrContains(destination, destinationIP) {
			continue
		}

		candidates = append(candidates, routeEntry{
			destination: destination,
			gateway:     stringValue(m["gateway"]),
			metric:      intValue(m["metric"], defaultRouteMetric),
			prefixLen:   prefixLen(destination),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].prefixLen != candidates[j].prefixLen {
			return candidates[i].prefixLen > candidates[j].prefixLen
		}

		return candidates[i].metric < candidates[j].metric
	})

	return candidates
}

// findNodeByAnyIP ищет узел по основному IP или IP любого интерфейса.
// Нужен для шлюзов: у роутера свой адрес в каждой подсети.
func (r *runner) findNodeByAnyIP(ip string) (model.Node, bool) {
	for _, node := range r.doc.Nodes {
		if nodeIP(node) == ip {
			return node, true
		}

		for _, iface := range anySlice(node.Config["interfaces"]) {
			if stringValue(anyMap(iface)["ip"]) == ip {
				return node, true
			}
		}
	}

	return model.Node{}, false
}

// sameSubnet сообщает, подключена ли подсеть с destinationIP к узлу напрямую:
// через его config.cidr или cidr одного из интерфейсов.
func sameSubnet(node model.Node, destinationIP string) bool {
	if cidr := stringValue(node.Config["cidr"]); cidr != "" && cidrContains(cidr, destinationIP) {
		return true
	}

	for _, iface := range anySlice(node.Config["interfaces"]) {
		if cidr := stringValue(anyMap(iface)["cidr"]); cidr != "" && cidrContains(cidr, destinationIP) {
			return true
		}
	}

	return false
}

// cidrContains проверяет, входит ли IP в CIDR. Строка без маски сравнивается
// с IP как адрес хоста.
func cidrContains(cidr, ipValue string) bool {
	ip := net.ParseIP(ipValue)
	if ip == nil {
		return false
	}

	if !strings.Contains(cidr, "/") {
		return cidr == ipValue
	}

	_, network, err := net.ParseCIDR(cidr)

	return err == nil && network.Contains(ip)
}

// prefixLen — длина маски CIDR; адрес без маски — /32, кривой CIDR — 0.
func prefixLen(cidr string) int {
	if !strings.Contains(cidr, "/") {
		return hostPrefixLen
	}

	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0
	}

	ones, _ := network.Mask.Size()

	return ones
}

// appendPath склеивает пути, не повторяя общий узел на стыке:
// [a b] + [b c] = [a b c].
func appendPath(current, segment []string) []string {
	if len(segment) == 0 {
		return current
	}

	start := 0
	if len(current) > 0 && current[len(current)-1] == segment[0] {
		start = 1
	}

	return append(current, segment[start:]...)
}
