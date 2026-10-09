// Package engine — доменный сервис расчёта виртуальной симуляции сети.
//
// Движок не отправляет настоящих пакетов: по документу топологии он шаг за шагом
// «проводит» виртуальный пакет от клиента до цели — DNS, выбор маршрута,
// firewall, TCP/TLS, выбор сервера Load Balancer'ом — и на каждом шаге пишет
// событие (model.Event) с виртуальным временем.
//
// Место в цепочке: service/application/simulation (запуск пользователем)
// и service/domain/checker (проверка решения квеста) → Engine.Run.
// Движок не знает ни про HTTP, ни про PostgreSQL: на входе RunRequest,
// на выходе RunResult.
//
// Детерминированность — главное свойство движка. Задержки обработки, потери
// пакетов и выбор при отказе берутся из math/rand с seed запроса, поэтому
// одинаковые seed, топология и сценарий дают побайтно одинаковые события.
// Отсюда правило для правок: порядок обращений к генератору случайных чисел
// (processingDelay, failoverDelay, retryDelay, packetLostOnPath) менять нельзя —
// иначе у всех сохранённых seed поменяется результат.
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
)

const (
	// maxSimulationEvents — предел событий одного запуска. Защищает от
	// разрастания ответа и таблицы simulation_events на патологических топологиях.
	maxSimulationEvents = 10000

	// defaultSeed — seed, если запрос его не задал (0): с нулевым seed
	// все запуски без seed совпадали бы с явным seed 0.
	defaultSeed = 1
)

// Engine считает симуляцию. Состояния между запусками не хранит: всё состояние
// одного запуска живёт в runner, поэтому один Engine безопасно используют
// параллельные запросы.
type Engine struct {
	validator TopologyValidator
}

// New создаёт движок поверх валидатора топологии.
func New(validator TopologyValidator) *Engine {
	return &Engine{
		validator: validator,
	}
}

// Run проводит сценарий по топологии и возвращает события и сводку.
//
// Две разные «неудачи»:
//   - error — симуляцию нельзя даже начать: нет ID, топология не прошла
//     валидацию (errs.TopologyInvalidError с подробностями) или не читается;
//   - RunResult со статусом failed — симуляция прошла, но сеть не доставила
//     пакет (нет маршрута, firewall запретил, порт закрыт...). Это нормальный
//     результат: его события объясняют пользователю, где сломалось.
//
// ctx проверяется один раз, после расчёта: сам расчёт идёт в памяти
// и занимает миллисекунды, прерывать его на середине незачем.
func (e *Engine) Run(ctx context.Context, req model.RunRequest) (model.RunResult, error) {
	if req.Seed == 0 {
		req.Seed = defaultSeed
	}

	if req.SimulationID == "" {
		return model.RunResult{}, errors.New("simulation id is required")
	}

	validation := e.validator.ValidateRaw(req.Topology)
	if !validation.Valid {
		return model.RunResult{Status: model.SimulationStatusFailed, Seed: req.Seed}, errs.TopologyInvalidError{Validation: validation}
	}

	var doc model.Document
	if err := json.Unmarshal(req.Topology, &doc); err != nil {
		// Текст ошибки попадает в результат проверки квеста — оставлен на английском.
		return model.RunResult{}, fmt.Errorf("decode topology: %w", err)
	}

	r := newRunner(req, doc)

	r.emit(model.EventSimulationStarted, model.EventSeverityInfo, "simulation started", "", "", "", map[string]any{"seed": req.Seed})
	r.advance(r.processingDelay(1, 3))
	r.emit(model.EventTopologyValidated, model.EventSeverityInfo, "topology validated", "", "", "", map[string]any{"valid": true})

	// Без корректного источника пакет создать нельзя. Протокольный разбор
	// в этом случае не собирается: разбирать ещё нечего.
	if !r.validateSourceClient() {
		r.summary.Status = r.status
		return r.result(), nil
	}

	r.advance(1)
	r.createPacket()
	r.runScenario()

	select {
	case <-ctx.Done():
		r.fail(ctx.Err().Error())
	default:
	}

	if len(r.events) > maxSimulationEvents {
		r.events = r.events[:maxSimulationEvents]
		r.fail("simulation event limit exceeded")
	}

	r.summary.Status = r.status
	r.summary.ProtocolDetails = r.buildProtocolDetails()

	return r.result(), nil
}

// runScenario выбирает расчёт по типу сценария. failover_demo — тот же HTTPS,
// но с принудительным шагом failover даже без упавших узлов.
func (r *runner) runScenario() {
	switch r.req.Scenario.Type {
	case model.ScenarioDNSLookup:
		r.runDNSLookup()
	case model.ScenarioICMPPing:
		r.runPing(false)
	case model.ScenarioHTTPSRequest:
		r.runHTTPS(false)
	case model.ScenarioFailoverDemo:
		r.runHTTPS(true)
	default:
		r.fail("unsupported scenario type: " + r.req.Scenario.Type)
	}
}
