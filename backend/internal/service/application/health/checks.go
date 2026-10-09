package health

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/netquest/netquest/backend/internal/model"
)

const (
	// natsPingTimeout — срок PING к NATS в обычной проверке. Он свой, а не из ctx
	// запроса: зависший брокер не должен задерживать ответ /health/ready.
	natsPingTimeout = 2 * time.Second
	// deepCheckKeyTTL — сколько живёт тестовый ключ Redis, если удалить его не вышло.
	deepCheckKeyTTL = 5 * time.Second
	// deepCheckValue — что пишем и ожидаем прочитать в глубоких проверках.
	deepCheckValue = "ok"
)

// Ready — быстрые проверки: PING каждой зависимости.
func (s *service) Ready(ctx context.Context) model.HealthReport {
	checks := map[string]model.ComponentCheck{
		componentPostgres: s.checkPostgres(ctx),
		componentRedis:    s.checkRedis(ctx),
		componentNATS:     s.checkNATS(),
	}

	return summarize(checks, false)
}

// Deep — глубокие проверки: Redis и NATS проверяются записью и чтением.
// Срок задаёт вызывающий через ctx.
func (s *service) Deep(ctx context.Context) model.HealthReport {
	checks := map[string]model.ComponentCheck{
		componentPostgres: s.checkPostgres(ctx),
		componentRedis:    s.deepRedis(ctx),
		componentNATS:     s.deepNATS(ctx),
	}

	return summarize(checks, true)
}

// checkPostgres — Ping берёт соединение из пула и выполняет пустой запрос.
func (s *service) checkPostgres(ctx context.Context) model.ComponentCheck {
	start := time.Now()

	if s.postgres == nil {
		return checkFailed(start, "postgres pool is not configured")
	}

	if err := s.postgres.Ping(ctx); err != nil {
		return checkFailed(start, err.Error())
	}

	return checkOK(start)
}

// checkRedis — команда PING.
func (s *service) checkRedis(ctx context.Context) model.ComponentCheck {
	start := time.Now()

	if s.redis == nil {
		return checkFailed(start, "redis client is not configured")
	}

	if err := s.redis.Ping(ctx).Err(); err != nil {
		return checkFailed(start, err.Error())
	}

	return checkOK(start)
}

// checkNATS — PING по общему соединению (с переподключением, если его нет).
func (s *service) checkNATS() model.ComponentCheck {
	start := time.Now()

	if s.nats == nil {
		return checkFailed(start, "nats client is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), natsPingTimeout)
	defer cancel()

	if err := s.nats.Ping(ctx); err != nil {
		return checkFailed(start, err.Error())
	}

	return checkOK(start)
}

// deepRedis записывает уникальный ключ с TTL, читает его обратно и удаляет.
// Уникальный ключ — чтобы параллельные проверки не мешали друг другу.
func (s *service) deepRedis(ctx context.Context) model.ComponentCheck {
	start := time.Now()

	if s.redis == nil {
		return checkFailed(start, "redis client is not configured")
	}

	key := "netquest:health:" + uuid.NewString()

	if err := s.redis.Set(ctx, key, deepCheckValue, deepCheckKeyTTL).Err(); err != nil {
		return checkFailed(start, err.Error())
	}

	// Удаляем с context.WithoutCancel: ключ надо убрать, даже если срок
	// проверки истёк. Не вышло — ключ сам исчезнет по TTL.
	defer s.redis.Del(context.WithoutCancel(ctx), key)

	got, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return checkFailed(start, err.Error())
	}

	if got != deepCheckValue {
		return checkFailed(start, "redis read-after-write mismatch")
	}

	return checkOK(start)
}

// deepNATS публикует сообщение в уникальный subject и ждёт его же по подписке.
func (s *service) deepNATS(ctx context.Context) model.ComponentCheck {
	start := time.Now()

	if s.nats == nil {
		return checkFailed(start, "nats client is not configured")
	}

	subject := "netquest.health." + uuid.NewString()

	if err := s.nats.PublishAndConsume(ctx, subject, []byte("ping")); err != nil {
		if strings.Contains(err.Error(), "mismatch") {
			return checkFailed(start, "nats publish/consume payload mismatch")
		}

		return checkFailed(start, err.Error())
	}

	return checkOK(start)
}
