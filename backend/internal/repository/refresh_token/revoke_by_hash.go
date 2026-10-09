package refreshtoken

import (
	"context"
	"fmt"
	"time"
)

// RevokeByHash отзывает токен. COALESCE сохраняет время первого отзыва:
// повторный отзыв (двойной logout) ничего не меняет и ошибкой не считается.
func (r *repository) RevokeByHash(ctx context.Context, tokenHash string, now time.Time) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = COALESCE(revoked_at, $2), updated_at = now()
		WHERE token_hash = $1`

	if _, err := r.pool.Exec(ctx, query, tokenHash, now); err != nil {
		return fmt.Errorf("отозвать refresh-токен: %w", err)
	}

	return nil
}
