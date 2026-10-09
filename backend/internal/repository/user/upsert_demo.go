package user

import (
	"context"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// UpsertDemoUser создаёт общий demo-аккаунт или обновляет существующий.
//
// ON CONFLICT (email): demo-пользователь один на всех, и каждый demo-вход
// возвращает его же — с обновлёнными именем и ролью и снятой пометкой
// удаления. ID у конфликтующей строки остаётся прежним.
func (r *repository) UpsertDemoUser(ctx context.Context, user model.User) (model.User, error) {
	const query = `
		INSERT INTO users (id, email, password_hash, display_name, role, created_at, updated_at)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, now(), now())
		ON CONFLICT (email) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    role = EXCLUDED.role,
		    updated_at = now(),
		    deleted_at = NULL
		RETURNING ` + userColumns

	upserted, err := r.queryOne(ctx, query, user.ID, user.Email, user.PasswordHash, user.DisplayName, user.Role)
	if err != nil {
		return model.User{}, fmt.Errorf("создать или обновить demo-пользователя: %w", err)
	}

	return upserted, nil
}
