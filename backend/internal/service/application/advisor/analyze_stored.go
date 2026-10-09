package advisor

import (
	"context"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// AnalyzeStored диагностирует сохранённую версию топологии пользователя.
// Чужая или несуществующая версия → errs.ErrTopologyNotFound (404).
func (s *service) AnalyzeStored(ctx context.Context, userID, topologyID string, scenario *model.Scenario) ([]model.Issue, error) {
	topology, err := s.topologyReader.GetForOwner(ctx, topologyID, userID)
	if err != nil {
		return nil, fmt.Errorf("получить топологию: %w", err)
	}

	return s.AnalyzeRaw(topology.Data, scenario)
}
