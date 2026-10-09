package refreshtoken

import (
	"context"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// Create сохраняет новый токен. Пустые user agent и IP пишутся как NULL;
// IP приводится к типу inet — строка, которая не является IP, даст ошибку БД.
func (r *repository) Create(ctx context.Context, token model.RefreshToken) error {
	const query = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, user_agent, ip_address, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, '')::inet, $6, now(), now())`

	if _, err := r.pool.Exec(ctx, query, token.ID, token.UserID, token.TokenHash, token.UserAgent, token.IPAddress, token.ExpiresAt); err != nil {
		return fmt.Errorf("создать refresh-токен: %w", err)
	}

	return nil
}
