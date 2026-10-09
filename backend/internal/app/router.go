package app

import (
	"net/http"

	"github.com/netquest/netquest/backend/internal/config"
	"github.com/netquest/netquest/backend/internal/metrics"
	"github.com/netquest/netquest/backend/internal/middleware"
)

// Обработчики API глазами роутера. Типы обработчиков в API-пакетах
// неэкспортируемые (как в solution), поэтому роутер описывает только нужные
// ему методы — интерфейсы у потребителя.

type authHandlers interface {
	Register(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	Demo(w http.ResponseWriter, r *http.Request)
	Refresh(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	Me(w http.ResponseWriter, r *http.Request)
}

type projectHandlers interface {
	List(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	Get(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
}

type topologyHandlers interface {
	ListForProject(w http.ResponseWriter, r *http.Request)
	CreateForProject(w http.ResponseWriter, r *http.Request)
	Get(w http.ResponseWriter, r *http.Request)
	Validate(w http.ResponseWriter, r *http.Request)
}

type advisorHandlers interface {
	AnalyzeRaw(w http.ResponseWriter, r *http.Request)
	AnalyzeStored(w http.ResponseWriter, r *http.Request)
}

type questHandlers interface {
	List(w http.ResponseWriter, r *http.Request)
	Get(w http.ResponseWriter, r *http.Request)
	Start(w http.ResponseWriter, r *http.Request)
	GetAttempt(w http.ResponseWriter, r *http.Request)
	Check(w http.ResponseWriter, r *http.Request)
	Reset(w http.ResponseWriter, r *http.Request)
	RevealHint(w http.ResponseWriter, r *http.Request)
}

type simulationHandlers interface {
	Start(w http.ResponseWriter, r *http.Request)
	Get(w http.ResponseWriter, r *http.Request)
	Events(w http.ResponseWriter, r *http.Request)
	Stream(w http.ResponseWriter, r *http.Request)
}

type healthHandlers interface {
	Live(w http.ResponseWriter, r *http.Request)
	Ready(w http.ResponseWriter, r *http.Request)
	Deep(w http.ResponseWriter, r *http.Request)
}

// routes — всё, из чего собирается HTTP-обработчик сервиса.
type routes struct {
	auth        authHandlers
	projects    projectHandlers
	topologies  topologyHandlers
	advisor     advisorHandlers
	quests      questHandlers
	simulations simulationHandlers
	health      healthHandlers

	tokenParser middleware.AccessTokenParser
	rateLimit   middleware.Middleware
	metrics     *metrics.Metrics
}

// newHTTPHandler регистрирует маршруты в ServeMux и оборачивает его в цепочку
// middleware.
//
// Маршруты — шаблоны Go 1.22 ("МЕТОД /путь/{параметр}"): ServeMux сам сверяет
// метод и достаёт параметры пути (r.PathValue). Защищённые маршруты обёрнуты
// в RequireAuth по одному, а не всем mux'ом: health, вход, регистрация,
// refresh, logout и WebSocket доступны без access-токена.
//
// Порядок middleware — снаружи внутрь:
//
//	RequestID → Recover → SecureHeaders → CORS → BodyLimit → RequestTimeout
//	→ RequestLogger → rate limiter → метрики → ServeMux → [RequireAuth] → хендлер
//
// RequestID — первым, чтобы ID был у всех записей лога, включая панику.
// Recover — до остальных: паника в любом middleware ниже тоже станет 500.
// CORS — до лимитера: preflight OPTIONS отвечает 204 и не расходует лимит.
func newHTTPHandler(cfg *config.Config, rt routes) http.Handler {
	mux := http.NewServeMux()
	requireAuth := middleware.RequireAuth(rt.tokenParser)

	protected := func(handler http.HandlerFunc) http.Handler {
		return requireAuth(handler)
	}

	mux.HandleFunc("GET /health/live", rt.health.Live)
	mux.HandleFunc("GET /health/ready", rt.health.Ready)
	mux.HandleFunc("GET /health/deep", rt.health.Deep)
	mux.HandleFunc("GET /metrics", rt.metrics.Handler)

	mux.HandleFunc("POST /api/v1/auth/register", rt.auth.Register)
	mux.HandleFunc("POST /api/v1/auth/login", rt.auth.Login)
	mux.HandleFunc("POST /api/v1/auth/demo", rt.auth.Demo)
	mux.HandleFunc("POST /api/v1/auth/refresh", rt.auth.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", rt.auth.Logout)
	mux.Handle("GET /api/v1/auth/me", protected(rt.auth.Me))

	mux.Handle("GET /api/v1/projects", protected(rt.projects.List))
	mux.Handle("POST /api/v1/projects", protected(rt.projects.Create))
	mux.Handle("GET /api/v1/projects/{projectId}", protected(rt.projects.Get))
	mux.Handle("PATCH /api/v1/projects/{projectId}", protected(rt.projects.Update))
	mux.Handle("DELETE /api/v1/projects/{projectId}", protected(rt.projects.Delete))
	mux.Handle("GET /api/v1/projects/{projectId}/topologies", protected(rt.topologies.ListForProject))
	mux.Handle("POST /api/v1/projects/{projectId}/topologies", protected(rt.topologies.CreateForProject))

	mux.Handle("GET /api/v1/quests", protected(rt.quests.List))
	mux.Handle("GET /api/v1/quests/{questId}", protected(rt.quests.Get))
	mux.Handle("POST /api/v1/quests/{questId}/start", protected(rt.quests.Start))
	mux.Handle("GET /api/v1/quest-attempts/{attemptId}", protected(rt.quests.GetAttempt))
	mux.Handle("POST /api/v1/quest-attempts/{attemptId}/check", protected(rt.quests.Check))
	mux.Handle("POST /api/v1/quest-attempts/{attemptId}/reset", protected(rt.quests.Reset))
	mux.Handle("POST /api/v1/quest-attempts/{attemptId}/reveal-hint", protected(rt.quests.RevealHint))

	mux.Handle("GET /api/v1/topologies/{topologyId}", protected(rt.topologies.Get))
	mux.Handle("POST /api/v1/topologies/{topologyId}/validate", protected(rt.topologies.Validate))
	mux.Handle("POST /api/v1/topologies/analyze", protected(rt.advisor.AnalyzeRaw))
	mux.Handle("POST /api/v1/topologies/{topologyId}/analyze", protected(rt.advisor.AnalyzeStored))

	mux.Handle("POST /api/v1/simulations", protected(rt.simulations.Start))
	mux.Handle("GET /api/v1/simulations/{simulationId}", protected(rt.simulations.Get))
	mux.Handle("GET /api/v1/simulations/{simulationId}/events", protected(rt.simulations.Events))
	mux.HandleFunc("GET /api/v1/ws", rt.simulations.Stream)

	return middleware.Chain(mux,
		middleware.RequestID,
		middleware.Recover,
		middleware.SecureHeaders,
		middleware.CORS(middleware.CORSConfig{
			AllowedOrigins:   cfg.HTTP.CORSAllowedOrigins,
			AllowCredentials: true,
		}),
		middleware.BodyLimit(cfg.HTTP.MaxRequestBodyBytes),
		middleware.RequestTimeout(cfg.HTTP.RequestTimeout),
		middleware.RequestLogger,
		rt.rateLimit,
		rt.metrics.Middleware,
	)
}
