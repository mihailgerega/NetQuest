package server

import (
	"log/slog"
	"net/http"

	"github.com/netquest/netquest/backend/internal/advisor"
	"github.com/netquest/netquest/backend/internal/auth"
	"github.com/netquest/netquest/backend/internal/config"
	"github.com/netquest/netquest/backend/internal/observability"
	"github.com/netquest/netquest/backend/internal/projects"
	"github.com/netquest/netquest/backend/internal/quests"
	"github.com/netquest/netquest/backend/internal/ratelimit"
	"github.com/netquest/netquest/backend/internal/security"
	"github.com/netquest/netquest/backend/internal/simulation"
	"github.com/netquest/netquest/backend/internal/topology"
)

type RouterDependencies struct {
	Config      config.Config
	Logger      *slog.Logger
	JWT         *auth.JWTManager
	Health      *observability.HealthHandler
	Metrics     *observability.Metrics
	Auth        *auth.Handler
	Advisor     *advisor.Handler
	Projects    *projects.Handler
	Quests      *quests.Handler
	Topologies  *topology.Handler
	Simulations *simulation.Handler
	RateLimiter *ratelimit.Limiter
}

func NewRouter(deps RouterDependencies) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health/live", deps.Health.Live)
	mux.HandleFunc("GET /health/ready", deps.Health.Ready)
	mux.HandleFunc("GET /health/deep", deps.Health.Deep)
	if deps.Metrics != nil {
		mux.HandleFunc("GET /metrics", deps.Metrics.Handler)
	}

	requireAuth := auth.RequireAuth(deps.JWT)
	mux.HandleFunc("POST /api/v1/auth/register", deps.Auth.Register)
	mux.HandleFunc("POST /api/v1/auth/login", deps.Auth.Login)
	mux.HandleFunc("POST /api/v1/auth/demo", deps.Auth.Demo)
	mux.HandleFunc("POST /api/v1/auth/refresh", deps.Auth.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", deps.Auth.Logout)
	mux.Handle("GET /api/v1/auth/me", requireAuth(http.HandlerFunc(deps.Auth.Me)))

	mux.Handle("GET /api/v1/projects", requireAuth(http.HandlerFunc(deps.Projects.List)))
	mux.Handle("POST /api/v1/projects", requireAuth(http.HandlerFunc(deps.Projects.Create)))
	mux.Handle("GET /api/v1/projects/{projectId}", requireAuth(http.HandlerFunc(deps.Projects.Get)))
	mux.Handle("PATCH /api/v1/projects/{projectId}", requireAuth(http.HandlerFunc(deps.Projects.Update)))
	mux.Handle("DELETE /api/v1/projects/{projectId}", requireAuth(http.HandlerFunc(deps.Projects.Delete)))
	mux.Handle("GET /api/v1/projects/{projectId}/topologies", requireAuth(http.HandlerFunc(deps.Topologies.ListForProject)))
	mux.Handle("POST /api/v1/projects/{projectId}/topologies", requireAuth(http.HandlerFunc(deps.Topologies.CreateForProject)))

	mux.Handle("GET /api/v1/quests", requireAuth(http.HandlerFunc(deps.Quests.List)))
	mux.Handle("GET /api/v1/quests/{questId}", requireAuth(http.HandlerFunc(deps.Quests.Get)))
	mux.Handle("POST /api/v1/quests/{questId}/start", requireAuth(http.HandlerFunc(deps.Quests.Start)))
	mux.Handle("GET /api/v1/quest-attempts/{attemptId}", requireAuth(http.HandlerFunc(deps.Quests.GetAttempt)))
	mux.Handle("POST /api/v1/quest-attempts/{attemptId}/check", requireAuth(http.HandlerFunc(deps.Quests.Check)))
	mux.Handle("POST /api/v1/quest-attempts/{attemptId}/reset", requireAuth(http.HandlerFunc(deps.Quests.Reset)))
	mux.Handle("POST /api/v1/quest-attempts/{attemptId}/reveal-hint", requireAuth(http.HandlerFunc(deps.Quests.RevealHint)))

	mux.Handle("GET /api/v1/topologies/{topologyId}", requireAuth(http.HandlerFunc(deps.Topologies.Get)))
	mux.Handle("POST /api/v1/topologies/{topologyId}/validate", requireAuth(http.HandlerFunc(deps.Topologies.Validate)))
	mux.Handle("POST /api/v1/topologies/analyze", requireAuth(http.HandlerFunc(deps.Advisor.AnalyzeRaw)))
	mux.Handle("POST /api/v1/topologies/{topologyId}/analyze", requireAuth(http.HandlerFunc(deps.Advisor.AnalyzeStored)))

	mux.Handle("POST /api/v1/simulations", requireAuth(http.HandlerFunc(deps.Simulations.Start)))
	mux.Handle("GET /api/v1/simulations/{simulationId}", requireAuth(http.HandlerFunc(deps.Simulations.Get)))
	mux.Handle("GET /api/v1/simulations/{simulationId}/events", requireAuth(http.HandlerFunc(deps.Simulations.Events)))
	mux.HandleFunc("GET /api/v1/ws", deps.Simulations.Stream)

	middleware := []Middleware{
		RequestID,
		Recover(deps.Logger),
		security.SecureHeaders,
		security.CORS(security.CORSConfig{
			AllowedOrigins:   deps.Config.HTTP.CORSAllowedOrigins,
			AllowCredentials: true,
		}),
		BodyLimit(deps.Config.HTTP.MaxRequestBodyBytes),
		RequestTimeout(deps.Config.HTTP.RequestTimeout),
		RequestLogger(deps.Logger),
	}
	if deps.RateLimiter != nil {
		middleware = append(middleware, deps.RateLimiter.Middleware)
	}
	if deps.Metrics != nil {
		middleware = append(middleware, deps.Metrics.Middleware)
	}

	return Chain(mux, middleware...)
}
