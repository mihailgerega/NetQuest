// Package converter — мост между HTTP-контрактом (internal/contract/*/v1)
// и доменом: запрос → input сервиса, модель → DTO ответа.
//
// Конвертеры только перекладывают поля: ни валидации, ни значений по умолчанию —
// это правила сервиса.
package converter

import (
	authv1 "github.com/netquest/netquest/backend/internal/contract/auth/v1"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// ToRegisterInput переводит запрос регистрации во вход сервиса.
func ToRegisterInput(req authv1.RegisterRequest) input.RegisterInput {
	return input.RegisterInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	}
}

// ToLoginInput переводит запрос входа во вход сервиса.
func ToLoginInput(req authv1.LoginRequest) input.LoginInput {
	return input.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}
}

// UserToDTO переводит пользователя в профиль для ответа. PasswordHash
// наружу не уходит: в контракте для него нет поля, и это намеренно.
func UserToDTO(user model.User) authv1.User {
	return authv1.User{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Role:        user.Role,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		DeletedAt:   user.DeletedAt,
	}
}

// AuthTokensToDTO переводит выданные токены в ответ. Refresh-токен
// в ответ не попадает — он уходит в cookie (см. authv1.AuthResponse).
func AuthTokensToDTO(tokens model.AuthTokens) authv1.AuthResponse {
	return authv1.AuthResponse{
		User:        UserToDTO(tokens.User),
		AccessToken: tokens.AccessToken,
		TokenType:   tokens.TokenType,
		ExpiresIn:   tokens.ExpiresIn,
	}
}
