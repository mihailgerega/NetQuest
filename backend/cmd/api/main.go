// Точка входа NetQuest API.
//
// main делает только то, что должно случиться до сборки графа зависимостей:
// загружает конфиг из переменных окружения. Всё остальное — DI-контейнер,
// HTTP-сервер, graceful shutdown — живёт в internal/app.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/netquest/netquest/backend/internal/app"
	"github.com/netquest/netquest/backend/internal/config"
)

// main только переводит ошибку run в код выхода 1.
func main() {
	if err := run(); err != nil {
		slog.Error("NetQuest API завершился с ошибкой", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	return app.New(context.Background(), cfg).Run()
}
