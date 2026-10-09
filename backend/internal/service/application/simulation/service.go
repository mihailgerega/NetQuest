// Package simulation — сервисный слой запусков симуляции: запуск сценария над
// сохранённой версией топологии, чтение запуска и его событий.
//
// Место в цепочке: api/simulation/v1 → service/application/simulation →
// {service/application/project (владелец), repository/topology, domain/engine,
// repository/simulation, producer/simulation_event (NATS)}.
//
// Симуляция считается синхронно, в запросе: движок работает в памяти
// за миллисекунды, а клиенту сразу нужен результат для Timeline.
package simulation

import "github.com/netquest/netquest/backend/internal/metrics"

// service — сервис запусков симуляции.
type service struct {
	simulationRepository SimulationRepository
	projectAuthorizer    ProjectAuthorizer
	topologyReader       TopologyReader
	engine               SimulationEngine
	eventProducer        EventProducer
	metrics              *metrics.Metrics
}

// New создаёт сервис запусков симуляции.
func New(
	simulationRepository SimulationRepository,
	projectAuthorizer ProjectAuthorizer,
	topologyReader TopologyReader,
	engine SimulationEngine,
	eventProducer EventProducer,
	counters *metrics.Metrics,
) *service {
	return &service{
		simulationRepository: simulationRepository,
		projectAuthorizer:    projectAuthorizer,
		topologyReader:       topologyReader,
		engine:               engine,
		eventProducer:        eventProducer,
		metrics:              counters,
	}
}
