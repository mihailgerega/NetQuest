package refreshtoken

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/converter"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// FindActiveByHash ищет действующий токен: не отозван и не истёк к моменту now.
// Нет такого → errs.ErrRefreshTokenInvalid (401): клиенту всё равно, был ли
// токен отозван, истёк или его не было вовсе.
func (r *repository) FindActiveByHash(ctx context.Context, tokenHash string, now time.Time) (model.RefreshToken, error) {
	const query = `
		SELECT id::text AS id, user_id::text AS user_id, token_hash,
		       COALESCE(user_agent, '') AS user_agent, COALESCE(ip_address::text, '') AS ip_address,
		       expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2`

	rows, err := r.pool.Query(ctx, query, tokenHash, now)
	if err != nil {
		return model.RefreshToken{}, fmt.Errorf("найти refresh-токен: %w", err)
	}

	rec, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[record.RefreshToken])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.RefreshToken{}, errs.ErrRefreshTokenInvalid
		}

		return model.RefreshToken{}, fmt.Errorf("прочитать refresh-токен: %w", err)
	}

	return converter.RefreshTokenToModel(rec), nil
}
