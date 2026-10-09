// CLI миграций NetQuest: migrate [up|status].
//
//	up     — применить новые миграции (по умолчанию);
//	status — показать все миграции и применены ли они.
//
// В Docker-образе запускается перед API (CMD: /app/migrate up && /app/api).
// Результат — простой текст в stdout: его читает человек или скрипт деплоя.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/netquest/netquest/backend/internal/config"
	"github.com/netquest/netquest/backend/pkg/migrator"
	"github.com/netquest/netquest/backend/pkg/postgres"
)

const (
	// migrateTimeout — общий срок работы CLI: подключение и все миграции.
	migrateTimeout = 30 * time.Second
	// exitUsage — код выхода при неизвестной команде (как у CLI-утилит).
	exitUsage = 2
)

func main() {
	os.Exit(run())
}

// run выполняет команду и возвращает код выхода. Отдельно от main, чтобы
// отложенные Close/cancel успели выполниться до os.Exit.
func run() int {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("загрузить конфигурацию", "error", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), migrateTimeout)
	defer cancel()

	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{
		DSN:             cfg.Postgres.DSN,
		MaxConns:        cfg.Postgres.MaxConns,
		MinConns:        cfg.Postgres.MinConns,
		ConnMaxLifetime: cfg.Postgres.ConnMaxLifetime,
	})
	if err != nil {
		slog.Error("подключиться к PostgreSQL", "error", err)
		return 1
	}
	defer pool.Close()

	m := migrator.Migrator{Pool: pool, Dir: cfg.Migrations.Dir}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "up":
		if err := m.Up(ctx); err != nil {
			slog.Error("применить миграции", "error", err)
			return 1
		}

		_, _ = fmt.Fprintln(os.Stdout, "migrations applied")
	case "status":
		if err := printStatus(ctx, m); err != nil {
			slog.Error("получить статус миграций", "error", err)
			return 1
		}
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown migrate command %q\n", command)
		return exitUsage
	}

	return 0
}

// printStatus печатает миграции построчно: "000001 000001_init.up.sql applied".
func printStatus(ctx context.Context, m migrator.Migrator) error {
	migrations, applied, err := m.Status(ctx)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		state := "pending"
		if applied[migration.Version] {
			state = "applied"
		}

		_, _ = fmt.Fprintf(os.Stdout, "%06d %s %s\n", migration.Version, migration.Name, state)
	}

	return nil
}
