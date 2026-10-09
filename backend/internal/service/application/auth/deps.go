package auth

import (
	"context"
	"time"

	"github.com/netquest/netquest/backend/internal/model"
)

// Интерфейсы зависимостей объявлены здесь, у потребителя (DIP): сервис не
// импортирует ни pgx, ни golang-jwt — наоборот, репозитории и JWTManager неявно
// удовлетворяют этим интерфейсам. Моки: mocks/ (task mocks:gen).

// UserRepository — то, что сервису нужно от хранилища пользователей.
type UserRepository interface {
	// Create сохраняет пользователя; занятый email → errs.ErrEmailTaken.
	Create(ctx context.Context, user model.User) (model.User, error)
	// FindByEmail ищет пользователя для Login; нет такого → errs.ErrUserNotFound.
	FindByEmail(ctx context.Context, email string) (model.User, error)
	// FindByID ищет пользователя для Me и Refresh; нет такого → errs.ErrUserNotFound.
	FindByID(ctx context.Context, id string) (model.User, error)
	// UpsertDemoUser создаёт или обновляет общий demo-аккаунт.
	UpsertDemoUser(ctx context.Context, user model.User) (model.User, error)
}

// RefreshTokenRepository — то, что сервису нужно от хранилища refresh-токенов.
type RefreshTokenRepository interface {
	// Create сохраняет токен (в виде хеша).
	Create(ctx context.Context, token model.RefreshToken) error
	// FindActiveByHash ищет действующий токен; нет такого → errs.ErrRefreshTokenInvalid.
	FindActiveByHash(ctx context.Context, tokenHash string, now time.Time) (model.RefreshToken, error)
	// RevokeByHash отзывает токен; повторный отзыв — не ошибка.
	RevokeByHash(ctx context.Context, tokenHash string, now time.Time) error
}

// AccessTokenIssuer — выпуск access-токенов (JWT). Реализация — *security.JWTManager.
type AccessTokenIssuer interface {
	GenerateAccessToken(userID, email, role string, now time.Time) (string, error)
}
