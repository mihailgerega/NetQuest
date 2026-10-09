package engine

import (
	"fmt"
	"sort"

	"github.com/netquest/netquest/backend/internal/model"
)

// Алгоритмы выбора сервера Load Balancer'ом (config.algorithm).
const (
	algorithmRoundRobin       = "round_robin"
	algorithmLeastConnections = "least_connections"
)

// errNoHealthyBackends — текст провала, когда выбрать сервер не из чего.
// Сравнивается в errorCodeForMessage, поэтому вынесен в константу.
const errNoHealthyBackends = "Load balancer has no healthy backends available."

// lbBackendCandidate — сервер из пула Load Balancer'а до проверки здоровья.
type lbBackendCandidate struct {
	NodeID  string
	Enabled bool
}

// selectBackend выбирает сервер для запроса, пришедшего на Load Balancer.
//
// Шаги: собрать кандидатов (пул + автообнаружение) → отсеять неисправных
// с причиной для каждого → выбрать по алгоритму балансировщика.
// Нет ни одного исправного — симуляция проваливается.
func (r *runner) selectBackend(lb model.Node, port int) (model.Node, bool) {
	candidates := r.lbBackendCandidates(lb)
	if len(candidates) == 0 {
		r.emit(model.EventLBBackendUnhealthy, model.EventSeverityError, "load balancer has no backend pool", lb.ID, "", r.packetID, map[string]any{
			"healthyBackends": []string{},
			"skippedBackends": []model.BackendSkip{},
		})
		r.fail(errNoHealthyBackends)

		return model.Node{}, false
	}

	healthy, healthyIDs, skipped := r.filterBackends(lb, candidates, port)

	r.summary.HealthyBackends = healthyIDs
	r.summary.SkippedBackends = skipped

	if len(skipped) > 0 {
		r.emit(model.EventLBBackendUnhealthy, model.EventSeverityWarn, "load balancer skipped unavailable backend(s)", lb.ID, "", r.packetID, map[string]any{
			"healthyBackends": healthyIDs,
			"skippedBackends": skipped,
		})
	}

	if len(healthy) == 0 {
		r.emit(model.EventLBBackendUnhealthy, model.EventSeverityError, "load balancer has no healthy backends available", lb.ID, "", r.packetID, map[string]any{
			"healthyBackends": healthyIDs,
			"skippedBackends": skipped,
		})
		r.fail(errNoHealthyBackends)

		return model.Node{}, false
	}

	// Сортировка на месте: срез healthyIDs уже лежит в сводке и в details события
	// выше, и сериализуются они позже — уже в отсортированном виде.
	sort.Slice(healthy, func(i, j int) bool { return healthy[i].ID < healthy[j].ID })
	sort.Strings(healthyIDs)

	algorithm := defaultString(lb.Config["algorithm"], algorithmRoundRobin)
	selected := r.pickBackend(algorithm, healthy)

	reason := fmt.Sprintf("%s selected %s", algorithm, nodeName(selected))
	if len(skipped) > 0 {
		reason = fmt.Sprintf("%s; skipped %d unavailable backend(s)", reason, len(skipped))
	}

	r.emit(model.EventLBBackendSelected, model.EventSeverityInfo, "load balancer selected backend", lb.ID, selected.ID, r.packetID, map[string]any{
		"algorithm":             algorithm,
		"selectedBackendNodeId": selected.ID,
		"selectedBackendName":   nodeName(selected),
		"reason":                reason,
		"healthyBackends":       healthyIDs,
		"skippedBackends":       skipped,
	})

	// Порт выбранного сервера уже проверен в filterBackends; вызов нужен ради
	// события server.port.open и строки в Decisions.
	r.ensureServerPortOpen(selected, httpsProtocol, port, lb.ID, selected.ID)
	r.summary.Decisions = append(r.summary.Decisions, "Load balancer selected "+nodeName(selected)+": "+reason)

	return selected, true
}

// filterBackends делит кандидатов на исправных и пропущенных (с причиной).
//
// Проверки в порядке: ID задан → узел существует → это server → включён
// в пуле → не выключен → слушает нужный порт → до него есть активный путь
// от балансировщика. Первая неудачная проверка и есть причина пропуска.
func (r *runner) filterBackends(lb model.Node, candidates []lbBackendCandidate, port int) ([]model.Node, []string, []model.BackendSkip) {
	healthy := make([]model.Node, 0, len(candidates))
	healthyIDs := make([]string, 0, len(candidates))
	skipped := make([]model.BackendSkip, 0)

	for _, candidate := range candidates {
		nodeID := candidate.NodeID
		if nodeID == "" {
			skipped = append(skipped, model.BackendSkip{Reason: "backend nodeId is empty"})
			continue
		}

		node, ok := r.nodes[nodeID]
		if !ok {
			skipped = append(skipped, model.BackendSkip{NodeID: nodeID, Reason: "backend node does not exist"})
			continue
		}

		if reason := r.backendSkipReason(lb, node, candidate, port); reason != "" {
			skipped = append(skipped, model.BackendSkip{NodeID: nodeID, Name: nodeName(node), Reason: reason})
			continue
		}

		healthy = append(healthy, node)
		healthyIDs = append(healthyIDs, nodeID)
	}

	return healthy, healthyIDs, skipped
}

// backendSkipReason возвращает причину не выбирать существующий узел или "",
// если он годится. Поиск пути от балансировщика перезаписывает r.routeInfo.
func (r *runner) backendSkipReason(lb, node model.Node, candidate lbBackendCandidate, port int) string {
	switch {
	case node.Type != model.NodeTypeServer:
		return "backend node is not a server"
	case !candidate.Enabled:
		return "backend is disabled"
	case nodeDown(node):
		return "node is down"
	case !serverPortCheck(node, httpsProtocol, port).Open:
		return fmt.Sprintf("server port %s/%d is closed", httpsProtocol, port)
	}

	if _, _, ok := r.findPath(lb.ID, node.ID); !ok {
		return "no active path from load balancer"
	}

	return ""
}

// pickBackend выбирает сервер из исправных (уже отсортированных по ID).
//
//   - round_robin — по seed: у одного seed всегда один и тот же сервер,
//     а разные seed распределяются по пулу;
//   - least_connections — с наименьшим config.activeConnections, при равенстве — меньший ID;
//   - неизвестный алгоритм — первый по ID.
func (r *runner) pickBackend(algorithm string, healthy []model.Node) model.Node {
	switch algorithm {
	case algorithmRoundRobin:
		return healthy[int(r.req.Seed%int64(len(healthy)))]
	case algorithmLeastConnections:
		sort.SliceStable(healthy, func(i, j int) bool {
			left := intValue(healthy[i].Config["activeConnections"], 0)
			right := intValue(healthy[j].Config["activeConnections"], 0)

			if left == right {
				return healthy[i].ID < healthy[j].ID
			}

			return left < right
		})

		return healthy[0]
	default:
		return healthy[0]
	}
}

// lbBackendCandidates собирает кандидатов: сначала пул config.backends
// (повторы nodeId отбрасываются), затем — если включён
// autoDiscoverConnectedServers — серверы, соединённые с балансировщиком каналом.
//
// Флаг включённости в пуле: enabled, иначе устаревшее healthy, иначе true.
func (r *runner) lbBackendCandidates(lb model.Node) []lbBackendCandidate {
	candidates := make([]lbBackendCandidate, 0)
	seen := map[string]struct{}{}

	for _, backend := range anySlice(lb.Config["backends"]) {
		m := anyMap(backend)

		nodeID := stringValue(m["nodeId"])
		if nodeID != "" {
			if _, ok := seen[nodeID]; ok {
				continue
			}

			seen[nodeID] = struct{}{}
		}

		candidates = append(candidates, lbBackendCandidate{
			NodeID:  nodeID,
			Enabled: boolValue(m["enabled"], boolValue(m["healthy"], true)),
		})
	}

	if !boolValue(lb.Config["autoDiscoverConnectedServers"], false) {
		return candidates
	}

	return append(candidates, r.discoverConnectedServers(lb, seen)...)
}

// discoverConnectedServers находит серверы, соединённые с балансировщиком
// напрямую, которых ещё нет в пуле. Каждая находка — событие lb.backend.discovered.
func (r *runner) discoverConnectedServers(lb model.Node, seen map[string]struct{}) []lbBackendCandidate {
	discovered := make([]lbBackendCandidate, 0)

	for _, link := range r.doc.Links {
		if link.SourceNodeID != lb.ID && link.TargetNodeID != lb.ID {
			continue
		}

		otherID := link.TargetNodeID
		if otherID == lb.ID {
			otherID = link.SourceNodeID
		}

		if _, ok := seen[otherID]; ok {
			continue
		}

		node, ok := r.nodes[otherID]
		if !ok || node.Type != model.NodeTypeServer {
			continue
		}

		seen[otherID] = struct{}{}
		discovered = append(discovered, lbBackendCandidate{NodeID: otherID, Enabled: true})

		r.emit(model.EventLBBackendDiscovered, model.EventSeverityInfo, "load balancer discovered connected server", lb.ID, otherID, r.packetID, map[string]any{
			"backend": otherID,
			"name":    nodeName(node),
		})
	}

	return discovered
}
