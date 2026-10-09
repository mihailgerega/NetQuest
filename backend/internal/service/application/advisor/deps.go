package advisor

import (
	"context"
	"encoding/json"

	"github.com/netquest/netquest/backend/internal/model"
)

// Интерфейсы зависимостей объявлены здесь, у потребителя (DIP). Моки: mocks/ (task mocks:gen).

// TopologyReader — то, что советнику нужно от хранилища топологий:
// прочитать сохранённую версию с проверкой владельца.
type TopologyReader interface {
	// GetForOwner возвращает версию топологии; чужая или несуществующая → errs.ErrTopologyNotFound.
	GetForOwner(ctx context.Context, topologyID, ownerID string) (model.Topology, error)
}

// TopologyValidator — то, что советнику нужно от валидатора топологии.
type TopologyValidator interface {
	ValidateRaw(data json.RawMessage) model.ValidationResult
}
