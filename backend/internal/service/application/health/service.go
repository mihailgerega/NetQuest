// Package health — проверки зависимостей сервиса для /health/ready и /health/deep.
//
// Две глубины проверки:
//   - Ready — дешёвые PING к PostgreSQL, Redis и NATS. Сервис готов принимать
//     трафик, пока жив PostgreSQL: без Redis работает лимитер в памяти, без NATS
//     не публикуются события — это degraded, а не error;
//   - Deep — запись и чтение: SET/GET в Redis, публикация с получением в NATS.
//     Любая неудача — error: такую проверку зовут при диагностике, а не балансировщиком.
//
// Тексты ошибок зависимостей отдаются клиенту как есть — это диагностический эндпоинт.
package health

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/pkg/nats"
)

// Имена зависимостей — ключи в HealthReport.Checks.
const (
	componentPostgres = "postgres"
	componentRedis    = "redis"
	componentNATS     = "nats"
)

// service проверяет зависимости. Любая из них может быть nil (не настроена) —
// тогда её проверка падает с понятной причиной, а не паникой.
type service struct {
	postgres *pgxpool.Pool
	redis    *redis.Client
	nats     *nats.Client
}

// New создаёт сервис проверок над уже созданными клиентами зависимостей.
func New(postgres *pgxpool.Pool, redisClient *redis.Client, natsClient *nats.Client) *service {
	return &service{
		postgres: postgres,
		redis:    redisClient,
		nats:     natsClient,
	}
}

// summarize сводит проверки в общий статус.
//
// Обычная проверка: упал PostgreSQL — error, упало что-то другое — degraded.
// Глубокая: любая неудача — error.
func summarize(checks map[string]model.ComponentCheck, deep bool) model.HealthReport {
	status := model.HealthStatusOK

	for name, check := range checks {
		if check.Status == model.HealthStatusOK {
			continue
		}

		if name == componentPostgres || deep {
			status = model.HealthStatusError
			break
		}

		status = model.HealthStatusDegraded
	}

	return model.HealthReport{
		Status:    status,
		Checks:    checks,
		Timestamp: time.Now().UTC(),
	}
}

// checkOK — проверка прошла; время — от start до сейчас.
func checkOK(start time.Time) model.ComponentCheck {
	return model.ComponentCheck{Status: model.HealthStatusOK, LatencyMs: time.Since(start).Milliseconds()}
}

// checkFailed — проверка не прошла, с причиной.
func checkFailed(start time.Time, message string) model.ComponentCheck {
	return model.ComponentCheck{Status: model.HealthStatusError, LatencyMs: time.Since(start).Milliseconds(), Error: message}
}
