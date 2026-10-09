package config

import "time"

// defaultPostgresDSN — база из docker-compose.yml для локального запуска.
const defaultPostgresDSN = "postgres://netquest:netquest@localhost:5432/netquest?sslmode=disable" //nolint:gosec // G101: учётка локального docker-compose, не секрет

// postgresConfig — подключение к PostgreSQL и размер пула pgxpool.
type postgresConfig struct {
	// DSN — строка подключения целиком: в production её собирает docker compose
	// из POSTGRES_USER/POSTGRES_PASSWORD, поэтому отдельных полей нет.
	DSN string
	// MaxConns и MinConns — границы пула на один процесс API.
	MaxConns int32
	MinConns int32
	// ConnMaxLifetime — через сколько пересоздавать соединение, даже живое:
	// так пул переживает перезапуск PostgreSQL и смену адреса за DNS.
	ConnMaxLifetime time.Duration
}

func loadPostgresConfig(lookup LookupFunc) postgresConfig {
	return postgresConfig{
		DSN:             getString(lookup, "POSTGRES_DSN", defaultPostgresDSN),
		MaxConns:        int32(getInt(lookup, "POSTGRES_MAX_CONNS", 10)), //nolint:gosec // G115: размер пула — десятки, не миллиарды
		MinConns:        int32(getInt(lookup, "POSTGRES_MIN_CONNS", 1)),  //nolint:gosec // G115: см. выше
		ConnMaxLifetime: getDuration(lookup, "POSTGRES_CONN_MAX_LIFETIME", time.Hour),
	}
}
