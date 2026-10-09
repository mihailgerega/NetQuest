// Package logger — настройка глобального slog для NetQuest API.
//
// Остальной код пишет логи через пакетные функции slog (slog.InfoContext,
// slog.WarnContext, ...) и логгер по цепочке вызовов не передаёт: Init один раз
// подменяет логгер по умолчанию, и все вызовы slog.* дальше идут в него.
package logger

import (
	"log/slog"
	"os"
)

// Init делает логгером по умолчанию JSON-логгер в stdout с уровнем INFO.
//
// JSON, а не текст: stdout контейнера читает docker logs и сборщики логов,
// а им удобнее разбирать структурированные записи. Атрибуты service и env
// попадают в каждую запись — по ним логи разных окружений не перепутать.
//
// До вызова Init slog пишет текстом в stderr (так ведёт себя пакет по умолчанию):
// этим пользуются main при ошибке загрузки конфига и CLI миграций.
func Init(serviceName, env string) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	slog.SetDefault(slog.New(handler).With(
		slog.String("service", serviceName),
		slog.String("env", env),
	))
}
