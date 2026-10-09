package v1

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// SimulationService — то, что обработчикам нужно от сервиса симуляций.
// Реализация — *service из service/application/simulation. Мок: mocks/ (task mocks:gen).
type SimulationService interface {
	Start(ctx context.Context, userID string, in input.StartSimulationInput) (model.Simulation, model.RunResult, error)
	Get(ctx context.Context, userID, simulationID string) (model.Simulation, error)
	Events(ctx context.Context, userID, simulationID string) ([]model.Event, error)
}

// AccessTokenParser проверяет access-токен WebSocket-клиента.
// Реализация — *security.JWTManager.
type AccessTokenParser interface {
	ParseAccessToken(raw string) (model.Principal, error)
}
