// Package converter — перевод строк PostgreSQL (repository/record) в доменные
// модели (model). В обратную сторону конвертеры не нужны: репозитории пишут
// поля модели параметрами запроса напрямую.
package converter

import (
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// UserToModel переводит строку users в доменную модель.
func UserToModel(rec record.User) model.User {
	return model.User{
		ID:           rec.ID,
		Email:        rec.Email,
		DisplayName:  rec.DisplayName,
		AvatarURL:    rec.AvatarURL,
		Role:         rec.Role,
		PasswordHash: rec.PasswordHash,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
		DeletedAt:    rec.DeletedAt,
	}
}

// RefreshTokenToModel переводит строку refresh_tokens в доменную модель.
func RefreshTokenToModel(rec record.RefreshToken) model.RefreshToken {
	return model.RefreshToken{
		ID:        rec.ID,
		UserID:    rec.UserID,
		TokenHash: rec.TokenHash,
		UserAgent: rec.UserAgent,
		IPAddress: rec.IPAddress,
		ExpiresAt: rec.ExpiresAt,
		RevokedAt: rec.RevokedAt,
		CreatedAt: rec.CreatedAt,
	}
}
