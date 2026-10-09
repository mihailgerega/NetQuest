package engine

import (
	"fmt"
	"math/rand" //nolint:depguard // нужен именно v1: последовательность rand.NewSource(seed) — часть сохранённых результатов, v2 даёт другую
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/pkg/idgen"
)

// runner — состояние одного запуска симуляции.
//
// Живёт ровно один вызов Run и в одной горутине, поэтому без мьютексов.
// Методы runner'а меняют его поля по ходу сценария: двигают виртуальные часы
// (timestamp), дописывают события и сводку.
type runner struct {
	req   model.RunRequest
	doc   model.Document
	nodes map[string]model.Node // узлы по ID для поиска за O(1)

	// rng — генератор с seed запроса. Единственный источник «случайности»
	// в расчёте: см. комментарий к пакету о порядке обращений к нему.
	rng *rand.Rand

	events    []model.Event
	timestamp int64 // виртуальное время в миллисекундах от начала симуляции
	packetID  string
	status    model.SimulationStatus
	summary   model.Summary

	// routeInfo — объяснение последнего поиска маршрута (findPath): почему
	// выбран путь или почему его нет. Попадает в события route.selected
	// и route.not_found. Перезаписывается каждым findPath.
	routeInfo string
}

// newRunner готовит состояние запуска. Статус по умолчанию — completed:
// провалить симуляцию может только явный fail.
func newRunner(req model.RunRequest, doc model.Document) *runner {
	nodes := make(map[string]model.Node, len(doc.Nodes))
	for _, node := range doc.Nodes {
		nodes[node.ID] = node
	}

	return &runner{
		req:    req,
		doc:    doc,
		nodes:  nodes,
		rng:    rand.New(rand.NewSource(req.Seed)), //nolint:gosec // G404: криптостойкость не нужна, нужна воспроизводимость по seed
		status: model.SimulationStatusCompleted,
		summary: model.Summary{
			Scenario:     req.Scenario.Type,
			Status:       model.SimulationStatusCompleted,
			Seed:         req.Seed,
			Source:       req.Scenario.SourceNodeID,
			SourceNodeID: req.Scenario.SourceNodeID,
			Destination:  req.Scenario.Target,
			// Пустые срезы, а не nil: клиент получает [], а не null.
			Decisions: []string{},
			Errors:    []string{},
			Path:      []string{},
		},
	}
}

// result собирает итог запуска из текущего состояния.
func (r *runner) result() model.RunResult {
	return model.RunResult{
		Status:  r.status,
		Seed:    r.req.Seed,
		Events:  r.events,
		Summary: r.summary,
	}
}

// validateSourceClient проверяет источник пакета: он задан, существует,
// это client и он не выключен. При ошибке проваливает симуляцию.
//
// Дальше сценарии берут источник из r.req.Scenario.SourceNodeID как есть,
// без обрезки пробелов: проверка здесь — только по обрезанному ID.
func (r *runner) validateSourceClient() bool {
	sourceID := strings.TrimSpace(r.req.Scenario.SourceNodeID)
	if sourceID == "" {
		r.fail("sourceNodeId is required")
		return false
	}

	source, ok := r.nodes[sourceID]
	if !ok {
		r.fail("source node does not exist")
		return false
	}

	if source.Type != model.NodeTypeClient {
		r.fail("source node must be a client")
		return false
	}

	if nodeDown(source) {
		r.fail("source client is down")
		return false
	}

	r.summary.Source = nodeName(source)
	r.summary.SourceNodeID = source.ID

	return true
}

// createPacket создаёт виртуальный пакет: ID берётся из генератора, поэтому
// он тоже воспроизводим по seed.
func (r *runner) createPacket() {
	r.packetID = fmt.Sprintf("pkt_%016x", r.rng.Uint64())
	r.summary.PacketID = r.packetID

	r.emit(model.EventPacketCreated, model.EventSeverityInfo, "packet created", r.req.Scenario.SourceNodeID, r.req.Scenario.Target, r.packetID, map[string]any{
		"scenario": r.req.Scenario.Type,
	})
	r.advance(1)
}

// complete завершает успешную симуляцию событием simulation.completed.
// После fail ничего не делает: провал окончателен.
//
// Сводка копируется в details по значению на этот момент — протокольного
// разбора в ней ещё нет, его Run соберёт позже.
func (r *runner) complete(message string) {
	if r.status == model.SimulationStatusFailed {
		return
	}

	if r.summary.TotalLatencyMs == 0 {
		r.summary.TotalLatencyMs = r.timestamp
	}

	r.emit(model.EventSimulationCompleted, model.EventSeverityInfo, "simulation completed", "", "", r.packetID, map[string]any{
		"summary": r.summary,
		"message": message,
	})
}

// fail проваливает симуляцию: статус failed, текст ошибки и её код — в сводку,
// событие simulation.failed — в Timeline. Срабатывает один раз: первая ошибка
// и есть причина, следующие вызовы ничего не меняют.
func (r *runner) fail(message string) {
	if r.status == model.SimulationStatusFailed {
		return
	}

	r.status = model.SimulationStatusFailed
	r.summary.Status = model.SimulationStatusFailed

	if r.summary.TotalLatencyMs == 0 {
		r.summary.TotalLatencyMs = r.timestamp
	}

	code := errorCodeForMessage(message)
	if code != "" {
		if r.summary.Metadata == nil {
			r.summary.Metadata = map[string]any{}
		}

		r.summary.Metadata["errorCode"] = code
	}

	r.summary.Errors = append(r.summary.Errors, message)

	r.emit(model.EventSimulationFailed, model.EventSeverityError, "simulation failed", "", "", r.packetID, map[string]any{
		"error":   message,
		"code":    code,
		"summary": r.summary,
	})
}

// emit дописывает событие с текущим виртуальным временем.
//
// ID события детерминирован: он строится из ID симуляции и номера события,
// поэтому повторный расчёт той же симуляции даёт те же ID.
func (r *runner) emit(
	eventType model.EventType,
	severity model.EventSeverity,
	message, source, target, packetID string,
	details map[string]any,
) {
	sequence := int64(len(r.events) + 1)

	if details == nil {
		details = map[string]any{}
	}

	r.events = append(r.events, model.Event{
		ID:             eventID(r.req.SimulationID, sequence),
		SimulationID:   r.req.SimulationID,
		SequenceNumber: sequence,
		Type:           eventType,
		TimestampMs:    r.timestamp,
		SourceNodeID:   source,
		TargetNodeID:   target,
		PacketID:       packetID,
		Severity:       severity,
		Message:        message,
		Details:        details,
	})
}

// addLatencyStage дописывает этап в разбор задержки.
func (r *runner) addLatencyStage(stage, label string, durationMs int64, details map[string]any) {
	if durationMs < 0 {
		durationMs = 0
	}

	if details == nil {
		details = map[string]any{}
	}

	r.summary.LatencyBreakdown = append(r.summary.LatencyBreakdown, model.LatencyStage{
		Stage:      stage,
		Label:      label,
		DurationMs: durationMs,
		Details:    details,
	})
}

// advance двигает виртуальные часы вперёд. Назад время не идёт:
// отрицательная задержка (кривой latencyMs в топологии) считается нулём.
func (r *runner) advance(ms int64) {
	if ms < 0 {
		ms = 0
	}

	r.timestamp += ms
}

// processingDelay — время обработки на узле: случайное в [minMs, maxMs].
func (r *runner) processingDelay(minMs, maxMs int64) int64 {
	if maxMs <= minMs {
		return minMs
	}

	return minMs + r.rng.Int63n(maxMs-minMs+1)
}

// failoverDelay — накладные расходы на переключение маршрута: 20–200 мс.
func (r *runner) failoverDelay() int64 {
	return 20 + r.rng.Int63n(181)
}

// retryDelay — пауза перед повторной отправкой потерянного пакета: 20–50 мс.
func (r *runner) retryDelay() int64 {
	return 20 + r.rng.Int63n(31)
}

// eventID строит детерминированный ID события из ID симуляции и номера события.
func eventID(simulationID string, sequence int64) string {
	return idgen.DeterministicUUID(fmt.Sprintf("%s_evt_%04d", simulationID, sequence))
}
