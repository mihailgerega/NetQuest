package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/netquest/netquest/backend/internal/config"
	"github.com/netquest/netquest/backend/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	postgres, err := storage.NewPostgres(ctx, cfg.Postgres)
	if err != nil {
		slog.Error("initialize postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer postgres.Close()

	migrator := storage.Migrator{Pool: postgres, Dir: cfg.Migrations.Dir}
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "up":
		if err := migrator.Up(ctx); err != nil {
			slog.Error("run migrations", slog.String("error", err.Error()))
			os.Exit(1)
		}
		fmt.Println("migrations applied")
	case "status":
		migrations, applied, err := migrator.Status(ctx)
		if err != nil {
			slog.Error("migration status", slog.String("error", err.Error()))
			os.Exit(1)
		}
		for _, migration := range migrations {
			state := "pending"
			if applied[migration.Version] {
				state = "applied"
			}
			fmt.Printf("%06d %s %s\n", migration.Version, migration.Name, state)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown migrate command %q\n", command)
		os.Exit(2)
	}
}
