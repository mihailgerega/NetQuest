package engine

import "github.com/netquest/netquest/backend/internal/model"

// Шаги, общие для ping и HTTPS: найти маршрут, при отказах пройти failover,
// зафиксировать выбранный маршрут и пережить потерю пакета.

// routeOrFail ищет маршрут до цели; если его нет — событие route.not_found
// с объяснением и провал симуляции.
func (r *runner) routeOrFail(source, targetID string) ([]string, int64, bool) {
	path, latency, ok := r.findPath(source, targetID)
	if !ok {
		r.emit(model.EventRouteNotFound, model.EventSeverityError, "route not found", source, targetID, r.packetID, map[string]any{
			"source":      source,
			"target":      targetID,
			"explanation": r.routeInfo,
		})
		r.fail("no route from " + source + " to " + targetID)

		return nil, 0, false
	}

	return path, latency, true
}

// failoverIfNeeded добавляет шаг failover, если его просит сценарий
// (failover_demo) или в топологии есть выключенный узел или канал.
// Маршрут к этому моменту уже найден в обход отказов — шаг показывает
// пользователю, что сеть перестроилась, и сколько это стоило по времени.
func (r *runner) failoverIfNeeded(forced bool, source, targetID string, path []string) {
	if !forced && !r.hasDownInfrastructure() {
		return
	}

	r.summary.Failover = true
	r.emit(model.EventFailoverTriggered, model.EventSeverityWarn, "failover triggered", source, targetID, r.packetID, map[string]any{
		"reason": "down node or link detected",
	})

	start := r.timestamp
	delay := r.failoverDelay()
	r.advance(delay)
	r.addLatencyStage("failover_overhead", "Failover overhead", r.timestamp-start, map[string]any{"processingMs": delay})

	r.emit(model.EventFailoverRouteChanged, model.EventSeverityInfo, "route recomputed after failure", source, targetID, r.packetID, map[string]any{
		"path": path,
	})
}

// selectRoute фиксирует выбранный маршрут: событие route.selected
// и этап «Route lookup» в разборе задержки.
func (r *runner) selectRoute(source, targetID string, path []string, latency int64) {
	routeStart := r.timestamp
	routeProcessing := r.processingDelay(1, 3)
	r.advance(routeProcessing)

	r.emit(model.EventRouteSelected, model.EventSeverityInfo, "route selected", source, targetID, r.packetID, map[string]any{
		"path":        path,
		"latencyMs":   latency,
		"algorithm":   r.routingAlgorithm(source, targetID),
		"explanation": r.routeInfo,
	})
	r.addLatencyStage("route_lookup", "Route lookup", r.timestamp-routeStart, map[string]any{
		"path":            path,
		"oneWayLatencyMs": latency,
		"processingMs":    routeProcessing,
	})
}

// packetLossMessages — тексты событий о потере пакета: у ping и HTTPS они разные.
type packetLossMessages struct {
	firstDrop  string
	secondDrop string
}

// survivePacketLoss разыгрывает потерю пакета на каналах пути.
//
// Пакет теряется — ждём retryDelay и пробуем второй раз. Потерян и второй —
// симуляция проваливается. Обе попытки тратят виртуальное время на путь.
// Возвращает false, если симуляция провалена.
func (r *runner) survivePacketLoss(source, targetID string, path []string, latency int64, messages packetLossMessages) bool {
	if !r.packetLostOnPath(path) {
		return true
	}

	lossStart := r.timestamp
	r.advance(latency)
	r.emit(model.EventPacketDropped, model.EventSeverityWarn, messages.firstDrop, source, targetID, r.packetID, map[string]any{
		"path":    path,
		"attempt": 1,
	})

	retry := r.retryDelay()
	r.advance(retry)

	if r.packetLostOnPath(path) {
		r.advance(latency)
		r.addPacketLossStage(lossStart, retry, path)
		r.emit(model.EventPacketDropped, model.EventSeverityWarn, messages.secondDrop, source, targetID, r.packetID, map[string]any{
			"path":    path,
			"attempt": 2,
		})
		r.fail("packet dropped because link packetLossPercent matched deterministic seed")

		return false
	}

	r.addPacketLossStage(lossStart, retry, path)
	r.emit(model.EventPacketForwarded, model.EventSeverityInfo, "packet retry succeeded", source, targetID, r.packetID, map[string]any{
		"attempt": 2,
		"path":    path,
	})

	return true
}

// addPacketLossStage — этап «Packet loss/retry» в разборе задержки.
func (r *runner) addPacketLossStage(lossStart, retry int64, path []string) {
	r.addLatencyStage("packet_loss_retry", "Packet loss/retry", r.timestamp-lossStart, map[string]any{
		"attempts":     2,
		"retryDelayMs": retry,
		"path":         path,
	})
}

// packetLostOnPath решает, потерялся ли пакет на одном из каналов пути.
//
// Каналы проверяются в порядке документа; генератор вызывается только для
// каналов пути с packetLossPercent > 0 и только до первой потери — это тоже
// часть воспроизводимости по seed.
func (r *runner) packetLostOnPath(path []string) bool {
	for _, link := range r.doc.Links {
		if !linkInPath(link, path) {
			continue
		}

		loss := floatValue(link.Config["packetLossPercent"], 0)
		if loss > 0 && r.rng.Float64()*100 < loss {
			return true
		}
	}

	return false
}

// linkInPath сообщает, соединяет ли канал два соседних узла пути
// (в любом направлении — каналы ненаправленные).
func linkInPath(link model.Link, path []string) bool {
	for i := range len(path) - 1 {
		forward := link.SourceNodeID == path[i] && link.TargetNodeID == path[i+1]
		backward := link.TargetNodeID == path[i] && link.SourceNodeID == path[i+1]

		if forward || backward {
			return true
		}
	}

	return false
}
