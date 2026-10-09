// Package refreshtoken — репозиторий refresh-токенов поверх PostgreSQL.
//
// Место в цепочке: service/application/auth → repository/refresh_token →
// таблица refresh_tokens. В таблице лежит SHA-256 токена, а не сам токен:
// утечка базы не даёт обновить чужую сессию. Поиск — по хешу.
package refreshtoken

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// repository читает и пишет refresh-токены через общий пул соединений.
type repository struct {
	pool *pgxpool.Pool
}

// New создаёт репозиторий refresh-токенов поверх пула.
func New(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}
