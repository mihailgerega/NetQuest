package checker

import (
	"context"
	"encoding/json"

	"github.com/netquest/netquest/backend/internal/model"
)

// Интерфейсы зависимостей объявлены здесь, у потребителя (DIP): чекер не
// импортирует ни движок, ни валидатор — их типы неявно удовлетворяют интерфейсам.

// SimulationEngine — то, что чекеру нужно от движка симуляции.
type SimulationEngine interface {
	Run(ctx context.Context, req model.RunRequest) (model.RunResult, error)
}

// TopologyValidator — то, что чекеру нужно от валидатора топологии.
type TopologyValidator interface {
	ValidateRaw(data json.RawMessage) model.ValidationResult
}
