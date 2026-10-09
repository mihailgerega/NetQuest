// Package simulationevent — публикация событий симуляции в NATS.
//
// Место в цепочке: service/application/simulation → producer → NATS
// (subject netquest.simulations.<id>.events). Подписчики — будущие realtime-
// клиенты; сейчас фронтенд получает события по WebSocket и опросом, а NATS
// служит шиной для них. Публикация — best effort: сервис логирует ошибку
// и продолжает работу.
package simulationevent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// messageTypeEvent — тип сообщения в конверте; тот же, что у кадров WebSocket.
const messageTypeEvent = "simulation.event"

// Message — конверт события в NATS: само событие, ID симуляции и тип сообщения.
// Поля в алфавитном порядке, как в прежнем сообщении (map[string]any).
type Message struct {
	Event        model.Event `json:"event"`
	SimulationID string      `json:"simulationId"`
	Type         string      `json:"type"`
}

// producer публикует события симуляции.
type producer struct {
	// publisher — nil, если NATS не настроен: тогда публикация ничего не делает.
	publisher Publisher
}

// New создаёт producer. publisher может быть nil — тогда события не публикуются.
func New(publisher Publisher) *producer {
	return &producer{
		publisher: publisher,
	}
}

// ProduceSimulationEvent публикует одно событие симуляции в её subject.
func (p *producer) ProduceSimulationEvent(ctx context.Context, simulationID string, event model.Event) error {
	if p.publisher == nil {
		return nil
	}

	payload, err := json.Marshal(Message{
		Event:        event,
		SimulationID: simulationID,
		Type:         messageTypeEvent,
	})
	if err != nil {
		return fmt.Errorf("сериализовать событие симуляции: %w", err)
	}

	if err := p.publisher.Publish(ctx, Subject(simulationID), payload); err != nil {
		return fmt.Errorf("опубликовать событие симуляции в NATS: %w", err)
	}

	return nil
}

// Subject — subject NATS для событий симуляции.
func Subject(simulationID string) string {
	return "netquest.simulations." + simulationID + ".events"
}
