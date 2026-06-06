package storage

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Dependencies struct {
	Postgres *pgxpool.Pool
	Redis    *redis.Client
	NATS     *NATSClient
	Logger   *slog.Logger
}

func (d Dependencies) Close() {
	if d.NATS != nil {
		d.NATS.Close()
	}
	if d.Redis != nil {
		if err := d.Redis.Close(); err != nil && d.Logger != nil {
			d.Logger.Error("close redis", slog.String("error", err.Error()))
		}
	}
	if d.Postgres != nil {
		d.Postgres.Close()
	}
}
