package storage

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/netquest/netquest/backend/pkg/idgen"
	"github.com/redis/go-redis/v9"
)

const (
	HealthStatusOK       = "ok"
	HealthStatusDegraded = "degraded"
	HealthStatusError    = "error"
)

type HealthChecker struct {
	Postgres *pgxpool.Pool
	Redis    *redis.Client
	NATS     *NATSClient
}

type HealthReport struct {
	Status    string                    `json:"status"`
	Checks    map[string]ComponentCheck `json:"checks"`
	Timestamp time.Time                 `json:"timestamp"`
}

type ComponentCheck struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latencyMs"`
	Error     string `json:"error,omitempty"`
}

func (h HealthChecker) Ready(ctx context.Context) HealthReport {
	checks := map[string]ComponentCheck{
		"postgres": h.checkPostgres(ctx),
		"redis":    h.checkRedis(ctx),
		"nats":     h.checkNATS(),
	}
	return summarize(checks, false)
}

func (h HealthChecker) Deep(ctx context.Context) HealthReport {
	checks := map[string]ComponentCheck{
		"postgres": h.checkPostgres(ctx),
		"redis":    h.deepRedis(ctx),
		"nats":     h.deepNATS(ctx),
	}
	return summarize(checks, true)
}

func (h HealthChecker) checkPostgres(ctx context.Context) ComponentCheck {
	start := time.Now()
	if h.Postgres == nil {
		return checkFailed(start, "postgres pool is not configured")
	}
	if err := h.Postgres.Ping(ctx); err != nil {
		return checkFailed(start, err.Error())
	}
	return checkOK(start)
}

func (h HealthChecker) checkRedis(ctx context.Context) ComponentCheck {
	start := time.Now()
	if h.Redis == nil {
		return checkFailed(start, "redis client is not configured")
	}
	if err := h.Redis.Ping(ctx).Err(); err != nil {
		return checkFailed(start, err.Error())
	}
	return checkOK(start)
}

func (h HealthChecker) checkNATS() ComponentCheck {
	start := time.Now()
	if h.NATS == nil {
		return checkFailed(start, "nats client is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := h.NATS.Ping(ctx); err != nil {
		return checkFailed(start, err.Error())
	}
	return checkOK(start)
}

func (h HealthChecker) deepRedis(ctx context.Context) ComponentCheck {
	start := time.Now()
	if h.Redis == nil {
		return checkFailed(start, "redis client is not configured")
	}

	id, err := idgen.NewUUID()
	if err != nil {
		return checkFailed(start, err.Error())
	}
	key := fmt.Sprintf("netquest:health:%s", id)
	value := "ok"
	if err := h.Redis.Set(ctx, key, value, 5*time.Second).Err(); err != nil {
		return checkFailed(start, err.Error())
	}
	defer h.Redis.Del(context.Background(), key)

	got, err := h.Redis.Get(ctx, key).Result()
	if err != nil {
		return checkFailed(start, err.Error())
	}
	if got != value {
		return checkFailed(start, "redis read-after-write mismatch")
	}
	return checkOK(start)
}

func (h HealthChecker) deepNATS(ctx context.Context) ComponentCheck {
	start := time.Now()
	if h.NATS == nil {
		return checkFailed(start, "nats client is not configured")
	}

	id, err := idgen.NewUUID()
	if err != nil {
		return checkFailed(start, err.Error())
	}
	subject := fmt.Sprintf("netquest.health.%s", id)
	payload := []byte("ping")
	if err := h.NATS.PublishAndConsume(ctx, subject, payload); err != nil {
		if bytes.Contains([]byte(err.Error()), []byte("mismatch")) {
			return checkFailed(start, "nats publish/consume payload mismatch")
		}
		return checkFailed(start, err.Error())
	}

	return checkOK(start)
}

func summarize(checks map[string]ComponentCheck, deep bool) HealthReport {
	status := HealthStatusOK
	for name, check := range checks {
		if check.Status == HealthStatusOK {
			continue
		}
		if name == "postgres" || deep {
			status = HealthStatusError
			break
		}
		status = HealthStatusDegraded
	}

	return HealthReport{
		Status:    status,
		Checks:    checks,
		Timestamp: time.Now().UTC(),
	}
}

func checkOK(start time.Time) ComponentCheck {
	return ComponentCheck{Status: HealthStatusOK, LatencyMs: time.Since(start).Milliseconds()}
}

func checkFailed(start time.Time, message string) ComponentCheck {
	return ComponentCheck{Status: HealthStatusError, LatencyMs: time.Since(start).Milliseconds(), Error: message}
}
