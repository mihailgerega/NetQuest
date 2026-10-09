package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/security"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// Register создаёт пользователя и сразу выдаёт ему токены — после регистрации
// отдельный вход не нужен.
//
// Шаги: нормализовать и проверить email → взять имя из email, если не задано →
// проверить длину пароля и захешировать его → INSERT в users → выдать токены.
//
// Занятость email заранее не проверяем (SELECT перед INSERT): два параллельных
// Register оба увидели бы «свободно». Гонку разрешает UNIQUE-ограничение в БД —
// второй INSERT упадёт, и репозиторий вернёт ErrEmailTaken.
func (s *service) Register(ctx context.Context, in input.RegisterInput, client input.ClientInfo) (model.AuthTokens, error) {
	email := normalizeEmail(in.Email)
	if email == "" {
		return model.AuthTokens{}, errs.NewValidationError("email is required", nil)
	}

	if !strings.Contains(email, "@") {
		return model.AuthTokens{}, errs.NewValidationError("email is invalid", nil)
	}

	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = strings.Split(email, "@")[0]
	}

	// Ошибки HashPassword (короткий пароль, больше 72 байт) — ошибки клиента:
	// их текст уходит в ответ 422 как есть.
	hash, err := security.HashPassword(in.Password, s.settings.PasswordHashCost)
	if err != nil {
		return model.AuthTokens{}, errs.NewValidationError(err.Error(), nil)
	}

	user, err := s.userRepository.Create(ctx, model.User{
		ID:           uuid.NewString(), // ID генерирует сервис: в таблице нет DEFAULT
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
		Role:         model.RoleUser,
	})
	if err != nil {
		// Занятый email — понятная клиенту ошибка, её отдаём без обёртки.
		if errors.Is(err, errs.ErrEmailTaken) {
			return model.AuthTokens{}, err
		}

		return model.AuthTokens{}, fmt.Errorf("сохранить пользователя: %w", err)
	}

	return s.issueTokens(ctx, user, client)
}

// normalizeEmail приводит email к виду, в котором он хранится и ищется:
// без пробелов по краям и в нижнем регистре.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
