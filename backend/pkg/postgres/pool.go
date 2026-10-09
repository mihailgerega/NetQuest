// Package postgres — создание пула соединений PostgreSQL (pgxpool) с размерами
// из конфига. Общий для API и CLI миграций.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig — параметры пула.
type PoolConfig struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	ConnMaxLifetime time.Duration
}

// NewPool создаёт пул. Соединения pgxpool открывает лениво (кроме MinConns,
// которые добирает в фоне), поэтому недоступная база здесь ошибкой не станет —
// её покажет первый запрос или health-check. Ошибка — только кривой DSN.
//
// pgxpool.Pool потокобезопасен: один пул на процесс, им пользуются все запросы.
func NewPool(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("разобрать DSN PostgreSQL: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("создать пул PostgreSQL: %w", err)
	}

	return pool, nil
}
