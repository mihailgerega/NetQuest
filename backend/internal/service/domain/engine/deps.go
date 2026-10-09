package engine

import (
	"encoding/json"

	"github.com/netquest/netquest/backend/internal/model"
)

// TopologyValidator — то, что движку нужно от валидатора топологии.
// Реализация — *validator.Validator; интерфейс объявлен здесь, у потребителя.
type TopologyValidator interface {
	ValidateRaw(data json.RawMessage) model.ValidationResult
}
