package v1

import (
	"context"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// AuthService — то, что обработчикам нужно от сервиса аутентификации.
//
// Интерфейс объявлен у потребителя (api), реализация — *service из
// service/application/auth (удовлетворяет неявно). Мок: mocks/ (task mocks:gen).
type AuthService interface {
	Register(ctx context.Context, in input.RegisterInput, client input.ClientInfo) (model.AuthTokens, error)
	Login(ctx context.Context, in input.LoginInput, client input.ClientInfo) (model.AuthTokens, error)
	Demo(ctx context.Context, client input.ClientInfo) (model.AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string, client input.ClientInfo) (model.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID string) (model.User, error)
}
