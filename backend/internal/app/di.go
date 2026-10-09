package app

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	advisorAPI "github.com/netquest/netquest/backend/internal/api/advisor/v1"
	"github.com/netquest/netquest/backend/internal/api/auditlog"
	authAPI "github.com/netquest/netquest/backend/internal/api/auth/v1"
	healthAPI "github.com/netquest/netquest/backend/internal/api/health/v1"
	projectAPI "github.com/netquest/netquest/backend/internal/api/project/v1"
	questAPI "github.com/netquest/netquest/backend/internal/api/quest/v1"
	simulationAPI "github.com/netquest/netquest/backend/internal/api/simulation/v1"
	topologyAPI "github.com/netquest/netquest/backend/internal/api/topology/v1"
	"github.com/netquest/netquest/backend/internal/config"
	"github.com/netquest/netquest/backend/internal/metrics"
	simulationEventProducer "github.com/netquest/netquest/backend/internal/producer/simulation_event"
	"github.com/netquest/netquest/backend/internal/ratelimit"
	auditRepository "github.com/netquest/netquest/backend/internal/repository/audit"
	projectRepository "github.com/netquest/netquest/backend/internal/repository/project"
	questRepository "github.com/netquest/netquest/backend/internal/repository/quest"
	refreshTokenRepository "github.com/netquest/netquest/backend/internal/repository/refresh_token"
	simulationRepository "github.com/netquest/netquest/backend/internal/repository/simulation"
	topologyRepository "github.com/netquest/netquest/backend/internal/repository/topology"
	userRepository "github.com/netquest/netquest/backend/internal/repository/user"
	"github.com/netquest/netquest/backend/internal/security"
	advisorService "github.com/netquest/netquest/backend/internal/service/application/advisor"
	auditService "github.com/netquest/netquest/backend/internal/service/application/audit"
	authService "github.com/netquest/netquest/backend/internal/service/application/auth"
	healthService "github.com/netquest/netquest/backend/internal/service/application/health"
	projectService "github.com/netquest/netquest/backend/internal/service/application/project"
	questService "github.com/netquest/netquest/backend/internal/service/application/quest"
	simulationService "github.com/netquest/netquest/backend/internal/service/application/simulation"
	topologyService "github.com/netquest/netquest/backend/internal/service/application/topology"
	"github.com/netquest/netquest/backend/internal/service/domain/catalog"
	"github.com/netquest/netquest/backend/internal/service/domain/checker"
	"github.com/netquest/netquest/backend/internal/service/domain/engine"
	"github.com/netquest/netquest/backend/internal/service/domain/validator"
	"github.com/netquest/netquest/backend/pkg/closer"
	"github.com/netquest/netquest/backend/pkg/nats"
	"github.com/netquest/netquest/backend/pkg/postgres"
)

// pgConnectTimeout — сколько ждать создания пула PostgreSQL при старте.
const pgConnectTimeout = 5 * time.Second

// projectServiceAPI — сервис проектов глазами контейнера: один объект
// обслуживает API проектов и проверку владельца для топологий и симуляций.
type projectServiceAPI interface {
	projectAPI.ProjectService
	topologyService.ProjectAuthorizer
}

// diContainer — контейнер зависимостей (Composition Root) NetQuest API.
//
// Граф (геттер слева вызывает геттеры справа):
//
//	AuthV1API       → AuthService       → UserRepository, RefreshTokenRepository → PGPool
//	                                    → JWTManager
//	ProjectV1API    → ProjectService    → ProjectRepository → PGPool
//	TopologyV1API   → TopologyService   → TopologyRepository, ProjectService, Validator
//	AdvisorV1API    → AdvisorService    → TopologyRepository, Validator
//	QuestV1API      → QuestService      → QuestRepository, Checker → Engine → Validator
//	SimulationV1API → SimulationService → SimulationRepository, ProjectService,
//	                                      TopologyRepository, Engine, EventProducer → NATSClient
//	HealthV1API     → HealthService     → PGPool, RedisClient, NATSClient
//	RateLimiter     → RedisClient
//
// Каждый геттер создаёт объект при первом вызове и дальше возвращает его же.
// Ресурсы (пул, Redis, NATS) регистрируют закрытие в closer в момент создания:
// порядок создания задаёт порядок закрытия (LIFO).
//
// Геттеры при ошибке логируют и завершают процесс (как в DI-примере курса).
// Контейнер не потокобезопасен: его заполняют один раз при старте, в одной горутине.
type diContainer struct {
	cfg *config.Config

	pgPool      *pgxpool.Pool
	redisClient *redis.Client
	// natsClient может остаться nil (кривой NATS_URL) — natsResolved отличает
	// «ещё не создавали» от «создать не вышло».
	natsClient   *nats.Client
	natsResolved bool

	jwtManager *security.JWTManager
	metrics    *metrics.Metrics
	validator  *validator.Validator
	engine     *engine.Engine

	userRepository         authService.UserRepository
	refreshTokenRepository authService.RefreshTokenRepository
	projectRepository      projectService.ProjectRepository
	topologyRepository     topologyService.TopologyRepository
	simulationRepository   simulationService.SimulationRepository
	questRepository        questService.QuestRepository

	auditService      auditlog.Recorder
	authService       authAPI.AuthService
	projectService    projectServiceAPI
	topologyService   topologyAPI.TopologyService
	advisorService    advisorAPI.AdvisorService
	questService      questAPI.QuestService
	simulationService simulationAPI.SimulationService
	healthService     healthAPI.HealthChecker
}

// newDIContainer создаёт пустой контейнер. Конфиг — его вход, а не зависимость:
// загружен и проверен в main ещё до сборки графа.
func newDIContainer(cfg *config.Config) *diContainer {
	return &diContainer{cfg: cfg}
}

// PGPool возвращает пул соединений PostgreSQL. При первом вызове создаёт пул
// и регистрирует его закрытие.
//
// Соединения pgxpool открывает лениво, и Ping здесь намеренно нет: API
// стартует и при недоступной базе (это видно в /health/ready), а запросы
// начнут проходить, как только база поднимется.
func (d *diContainer) PGPool(ctx context.Context) *pgxpool.Pool {
	if d.pgPool == nil {
		pool, err := openPostgres(ctx, d.cfg)
		if err != nil {
			slog.Error("создать пул PostgreSQL", "error", err)
			os.Exit(1)
		}

		closer.Add("пул PostgreSQL", func(context.Context) error {
			pool.Close()
			return nil
		})

		d.pgPool = pool
	}

	return d.pgPool
}

// RedisClient возвращает клиент Redis (лимитер и глубокий health-check).
// Соединение тоже ленивое: недоступный Redis не мешает старту — лимитер
// перейдёт на счётчик в памяти.
func (d *diContainer) RedisClient() *redis.Client {
	if d.redisClient == nil {
		options := &redis.Options{
			Addr:     d.cfg.Redis.Addr,
			Username: d.cfg.Redis.Username,
			Password: d.cfg.Redis.Password,
			DB:       d.cfg.Redis.DB,
		}

		if d.cfg.Redis.TLSEnabled {
			options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}

		client := redis.NewClient(options)

		closer.Add("Redis", func(context.Context) error {
			return client.Close()
		})

		d.redisClient = client
	}

	return d.redisClient
}

// NATSClient возвращает клиент NATS для событий симуляции.
//
// Недоступный брокер не останавливает сервис (degraded-режим): клиент
// возвращается и сам переподключится при следующей операции. А вот кривой
// NATS_URL даёт nil — тогда события не публикуются, а health покажет причину.
func (d *diContainer) NATSClient() *nats.Client {
	if !d.natsResolved {
		client, err := nats.New(d.cfg.NATS.URL, d.cfg.NATS.Timeout)
		if err != nil {
			slog.Warn("NATS недоступен, сервис работает в degraded-режиме", "error", err)
		}

		if client != nil {
			closer.Add("NATS", func(context.Context) error {
				client.Close()
				return nil
			})
		}

		d.natsClient = client
		d.natsResolved = true
	}

	return d.natsClient
}

// JWTManager возвращает выпуск и проверку access-токенов.
func (d *diContainer) JWTManager() *security.JWTManager {
	if d.jwtManager == nil {
		d.jwtManager = security.NewJWTManager(d.cfg.Security.JWTIssuer, d.cfg.Security.JWTSecret, d.cfg.Security.AccessTokenTTL)
	}

	return d.jwtManager
}

// Metrics возвращает счётчики сервиса.
func (d *diContainer) Metrics() *metrics.Metrics {
	if d.metrics == nil {
		d.metrics = metrics.New()
	}

	return d.metrics
}

// Validator возвращает валидатор топологии.
func (d *diContainer) Validator() *validator.Validator {
	if d.validator == nil {
		d.validator = validator.New()
	}

	return d.validator
}

// Engine возвращает движок симуляции.
func (d *diContainer) Engine() *engine.Engine {
	if d.engine == nil {
		d.engine = engine.New(d.Validator())
	}

	return d.engine
}

// RateLimiter возвращает лимитер запросов поверх Redis.
func (d *diContainer) RateLimiter() *ratelimit.Limiter {
	return ratelimit.New(d.RedisClient(), d.cfg.RateLimit.RequestsPerMinute, d.cfg.RateLimit.RedisPrefix)
}

// UserRepository возвращает репозиторий пользователей.
func (d *diContainer) UserRepository(ctx context.Context) authService.UserRepository {
	if d.userRepository == nil {
		d.userRepository = userRepository.New(d.PGPool(ctx))
	}

	return d.userRepository
}

// RefreshTokenRepository возвращает репозиторий refresh-токенов.
func (d *diContainer) RefreshTokenRepository(ctx context.Context) authService.RefreshTokenRepository {
	if d.refreshTokenRepository == nil {
		d.refreshTokenRepository = refreshTokenRepository.New(d.PGPool(ctx))
	}

	return d.refreshTokenRepository
}

// ProjectRepository возвращает репозиторий проектов.
func (d *diContainer) ProjectRepository(ctx context.Context) projectService.ProjectRepository {
	if d.projectRepository == nil {
		d.projectRepository = projectRepository.New(d.PGPool(ctx))
	}

	return d.projectRepository
}

// TopologyRepository возвращает репозиторий версий топологии. Его читают
// три сервиса: топологий, симуляций и советника.
func (d *diContainer) TopologyRepository(ctx context.Context) topologyService.TopologyRepository {
	if d.topologyRepository == nil {
		d.topologyRepository = topologyRepository.New(d.PGPool(ctx))
	}

	return d.topologyRepository
}

// SimulationRepository возвращает репозиторий симуляций и их событий.
func (d *diContainer) SimulationRepository(ctx context.Context) simulationService.SimulationRepository {
	if d.simulationRepository == nil {
		d.simulationRepository = simulationRepository.New(d.PGPool(ctx))
	}

	return d.simulationRepository
}

// QuestRepository возвращает репозиторий квестов и попыток.
func (d *diContainer) QuestRepository(ctx context.Context) questService.QuestRepository {
	if d.questRepository == nil {
		d.questRepository = questRepository.New(d.PGPool(ctx))
	}

	return d.questRepository
}

// AuditService возвращает сервис журнала аудита.
func (d *diContainer) AuditService(ctx context.Context) auditlog.Recorder {
	if d.auditService == nil {
		d.auditService = auditService.New(auditRepository.New(d.PGPool(ctx)))
	}

	return d.auditService
}

// AuthService возвращает сервис аутентификации.
func (d *diContainer) AuthService(ctx context.Context) authAPI.AuthService {
	if d.authService == nil {
		d.authService = authService.New(
			d.UserRepository(ctx),
			d.RefreshTokenRepository(ctx),
			d.JWTManager(),
			authService.Settings{
				AccessTokenTTL:   d.cfg.Security.AccessTokenTTL,
				RefreshTokenTTL:  d.cfg.Security.RefreshTokenTTL,
				PasswordHashCost: d.cfg.Security.PasswordHashCost,
				DemoAuthEnabled:  d.cfg.Security.DemoAuthEnabled,
			},
		)
	}

	return d.authService
}

// ProjectService возвращает сервис проектов.
func (d *diContainer) ProjectService(ctx context.Context) projectServiceAPI {
	if d.projectService == nil {
		d.projectService = projectService.New(d.ProjectRepository(ctx))
	}

	return d.projectService
}

// TopologyService возвращает сервис версий топологии.
func (d *diContainer) TopologyService(ctx context.Context) topologyAPI.TopologyService {
	if d.topologyService == nil {
		d.topologyService = topologyService.New(d.TopologyRepository(ctx), d.ProjectService(ctx), d.Validator())
	}

	return d.topologyService
}

// AdvisorService возвращает советника.
func (d *diContainer) AdvisorService(ctx context.Context) advisorAPI.AdvisorService {
	if d.advisorService == nil {
		d.advisorService = advisorService.New(d.TopologyRepository(ctx), d.Validator())
	}

	return d.advisorService
}

// QuestService возвращает сервис квестов над каталогом из кода.
func (d *diContainer) QuestService(ctx context.Context) questAPI.QuestService {
	if d.questService == nil {
		d.questService = questService.New(
			d.QuestRepository(ctx),
			checker.New(d.Engine(), d.Validator()),
			catalog.Catalog(),
		)
	}

	return d.questService
}

// SimulationService возвращает сервис запусков симуляции.
//
// Публикатор NATS передаётся интерфейсом, только если клиент создан: nil-указатель
// *nats.Client, завёрнутый в интерфейс, был бы «не nil», и producer вызвал бы
// у него метод с паникой. Без клиента producer просто не публикует.
func (d *diContainer) SimulationService(ctx context.Context) simulationAPI.SimulationService {
	if d.simulationService == nil {
		var publisher simulationEventProducer.Publisher
		if client := d.NATSClient(); client != nil {
			publisher = client
		}

		d.simulationService = simulationService.New(
			d.SimulationRepository(ctx),
			d.ProjectService(ctx),
			d.TopologyRepository(ctx),
			d.Engine(),
			simulationEventProducer.New(publisher),
			d.Metrics(),
		)
	}

	return d.simulationService
}

// HealthService возвращает проверки зависимостей.
func (d *diContainer) HealthService(ctx context.Context) healthAPI.HealthChecker {
	if d.healthService == nil {
		d.healthService = healthService.New(d.PGPool(ctx), d.RedisClient(), d.NATSClient())
	}

	return d.healthService
}

// Routes собирает обработчики всех API для роутера.
func (d *diContainer) Routes(ctx context.Context) routes {
	jsonLimit := d.cfg.HTTP.JSONBodyLimitBytes

	return routes{
		auth:        authAPI.New(d.AuthService(ctx), d.AuditService(ctx), jsonLimit, d.cfg.Security.SecureCookie),
		projects:    projectAPI.New(d.ProjectService(ctx), d.AuditService(ctx), jsonLimit),
		topologies:  topologyAPI.New(d.TopologyService(ctx), d.AuditService(ctx), jsonLimit),
		advisor:     advisorAPI.New(d.AdvisorService(ctx), d.AuditService(ctx), jsonLimit, d.Metrics()),
		quests:      questAPI.New(d.QuestService(ctx), d.AuditService(ctx), jsonLimit, d.Metrics()),
		simulations: simulationAPI.New(d.SimulationService(ctx), d.AuditService(ctx), d.JWTManager(), jsonLimit, d.Metrics()),
		health:      healthAPI.New(d.cfg.App.ServiceName, d.HealthService(ctx), d.cfg.Security.HealthDeepTimeout),
		tokenParser: d.JWTManager(),
		rateLimit:   d.RateLimiter().Middleware,
		metrics:     d.Metrics(),
	}
}

// openPostgres создаёт пул PostgreSQL с размерами из конфига. На создание
// даётся pgConnectTimeout: без срока зависшая сеть подвесила бы старт.
func openPostgres(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	startupCtx, cancel := context.WithTimeout(ctx, pgConnectTimeout)
	defer cancel()

	return postgres.NewPool(startupCtx, postgres.PoolConfig{
		DSN:             cfg.Postgres.DSN,
		MaxConns:        cfg.Postgres.MaxConns,
		MinConns:        cfg.Postgres.MinConns,
		ConnMaxLifetime: cfg.Postgres.ConnMaxLifetime,
	})
}
