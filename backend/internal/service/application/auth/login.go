package auth

import (
	"context"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/security"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// Login проверяет email и пароль и выдаёт токены.
//
// Любая неудача — нет такого email, у пользователя нет пароля (demo-аккаунт),
// пароль не подошёл и даже сбой поиска в базе — даёт одну ошибку
// ErrInvalidCredentials (401). По ответу нельзя узнать, зарегистрирован ли email.
func (s *service) Login(ctx context.Context, in input.LoginInput, client input.ClientInfo) (model.AuthTokens, error) {
	user, err := s.userRepository.FindByEmail(ctx, normalizeEmail(in.Email))
	if err != nil || user.PasswordHash == "" || !security.VerifyPassword(in.Password, user.PasswordHash) {
		return model.AuthTokens{}, errs.ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user, client)
}
