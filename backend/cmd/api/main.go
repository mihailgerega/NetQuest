package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/netquest/netquest/backend/internal/advisor"
	"github.com/netquest/netquest/backend/internal/audit"
	"github.com/netquest/netquest/backend/internal/auth"
	"github.com/netquest/netquest/backend/internal/config"
	"github.com/netquest/netquest/backend/internal/observability"
	"github.com/netquest/netquest/backend/internal/projects"
	"github.com/netquest/netquest/backend/internal/quests"
	"github.com/netquest/netquest/backend/internal/ratelimit"
	"github.com/netquest/netquest/backend/internal/realtime"
	"github.com/netquest/netquest/backend/internal/server"
	"github.com/netquest/netquest/backend/internal/simulation"
	"github.com/netquest/netquest/backend/internal/storage"
	"github.com/netquest/netquest/backend/internal/topology"
	"github.com/netquest/netquest/backend/internal/users"
	"github.com/netquest/netquest/backend/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log := logger.New(cfg.App.ServiceName, cfg.App.Env)
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	postgresCtx, cancelPostgres := context.WithTimeout(rootCtx, 5*time.Second)
	postgres, err := storage.NewPostgres(postgresCtx, cfg.Postgres)
	cancelPostgres()
	if err != nil {
		log.Error("initialize postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}

	redisClient := storage.NewRedis(cfg.Redis)
	natsConn, err := storage.NewNATS(cfg.NATS)
	if err != nil {
		log.Warn("initialize nats in degraded mode", slog.String("error", err.Error()))
	}

	deps := storage.Dependencies{
		Postgres: postgres,
		Redis:    redisClient,
		NATS:     natsConn,
		Logger:   log,
	}
	defer deps.Close()

	metrics := &observability.Metrics{}
	jwtManager := auth.NewJWTManager(cfg.Security)
	userRepo := users.NewRepository(postgres)
	refreshRepo := auth.NewRefreshRepository(postgres)
	authService := auth.NewService(userRepo, refreshRepo, jwtManager, cfg.Security)

	projectRepo := projects.NewRepository(postgres)
	projectService := projects.NewService(projectRepo)

	topologyValidator := topology.NewValidator()
	topologyRepo := topology.NewRepository(postgres)
	topologyService := topology.NewService(topologyRepo, projectService, topologyValidator)

	simulationRepo := simulation.NewRepository(postgres)
	simulationEngine := simulation.NewBasicEngine(topologyValidator)
	realtimePublisher := realtime.NewNATSPublisher(natsConn)
	questRepo := quests.NewRepository(postgres)
	questChecker := quests.NewChecker(simulationEngine, topologyValidator)
	questService := quests.NewService(questRepo, questChecker)
	advisorService := advisor.NewService(topologyRepo, topologyValidator)

	auditRepo := audit.NewRepository(postgres)
	auditService := audit.NewService(auditRepo)

	simulationService := simulation.NewService(
		simulationRepo,
		projectService,
		topologyRepo,
		simulationEngine,
		realtimePublisher,
		log,
		metrics,
	)

	healthHandler := &observability.HealthHandler{
		ServiceName: cfg.App.ServiceName,
		Checker: storage.HealthChecker{
			Postgres: postgres,
			Redis:    redisClient,
			NATS:     natsConn,
		},
		DeepTimeout: cfg.Security.HealthDeepTimeout,
	}

	router := server.NewRouter(server.RouterDependencies{
		Config:      cfg,
		Logger:      log,
		JWT:         jwtManager,
		Health:      healthHandler,
		Metrics:     metrics,
		Advisor:     advisor.NewHandler(advisorService, auditService, log, cfg.HTTP.JSONBodyLimitBytes, metrics),
		Auth:        auth.NewHandler(authService, auditService, log, cfg.HTTP.JSONBodyLimitBytes, cfg.Security.SecureCookie),
		Projects:    projects.NewHandler(projectService, auditService, log, cfg.HTTP.JSONBodyLimitBytes),
		Quests:      quests.NewHandler(questService, auditService, log, cfg.HTTP.JSONBodyLimitBytes, metrics),
		Topologies:  topology.NewHandler(topologyService, auditService, log, cfg.HTTP.JSONBodyLimitBytes),
		Simulations: simulation.NewHandler(simulationService, auditService, log, cfg.HTTP.JSONBodyLimitBytes, jwtManager, metrics),
		RateLimiter: ratelimit.New(redisClient, cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.RedisPrefix, log),
	})

	apiServer := server.New(cfg.HTTP, router, log)
	if err := apiServer.Run(rootCtx); err != nil {
		log.Error("api server stopped with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
