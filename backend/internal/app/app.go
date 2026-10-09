// Package app — сборка и жизненный цикл NetQuest API.
//
// di.go — DI-контейнер: что из чего создаётся. router.go — маршруты и цепочка
// middleware. app.go — порядок жизни процесса: инициализация → запуск
// HTTP-сервера → ожидание сигнала → закрытие ресурсов через closer
// в обратном порядке (LIFO).
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/netquest/netquest/backend/internal/config"
	"github.com/netquest/netquest/backend/pkg/closer"
	"github.com/netquest/netquest/backend/pkg/logger"
)

// App — корневая структура приложения: держит конфиг, DI-контейнер
// и то, что нужно для запуска HTTP-сервера.
type App struct {
	cfg         *config.Config
	diContainer *diContainer
	httpServer  *http.Server
	listener    net.Listener
}

// New создаёт приложение и инициализирует все зависимости.
// Ошибки инициализации (кривой DSN, занят порт) завершают процесс с логом.
func New(ctx context.Context, cfg *config.Config) *App {
	a := &App{cfg: cfg}
	a.initDeps(ctx)

	return a
}

// Run запускает HTTP-сервер и блокирует до сигнала SIGINT/SIGTERM или падения
// сервера, после чего синхронно закрывает ресурсы через closer.CloseAll.
func (a *App) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Буфер 1: горутина запишет результат Serve и завершится, даже если его уже не ждут.
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.runHTTPServer()
	}()

	var runErr error
	select {
	case runErr = <-errCh:
		// Сервер упал сам — всё равно закрываем то, что успели открыть.
	case <-ctx.Done():
		slog.Info("получен сигнал завершения, начинаем graceful shutdown")
	}
	// Снимаем перехват сигналов: повторный Ctrl+C завершит процесс сразу.
	cancel()

	// Новый контекст от Background: ctx уже отменён сигналом, а closer'ам нужен
	// живой срок — иначе Shutdown сразу оборвал бы текущие запросы.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()

	if err := closer.CloseAll(shutdownCtx); err != nil {
		slog.Error("ошибка при завершении работы", "error", err)

		if runErr == nil {
			runErr = err
		}
	}

	return runErr
}

// initDeps инициализирует зависимости по порядку: каждый шаг может опираться
// на результат предыдущего.
func (a *App) initDeps(ctx context.Context) {
	inits := []func(context.Context){
		a.initDI,
		a.initLogger,
		a.initStorage,
		a.initListener,
		a.initHTTPServer,
	}

	for _, initFn := range inits {
		initFn(ctx)
	}
}

// initDI создаёт пустой DI-контейнер: сами зависимости появятся при первом обращении.
func (a *App) initDI(context.Context) {
	a.diContainer = newDIContainer(a.cfg)
}

// initLogger делает логгером по умолчанию JSON-логгер с атрибутами service и env.
// До этого момента slog пишет текстом в stderr.
func (a *App) initLogger(context.Context) {
	logger.Init(a.cfg.App.ServiceName, a.cfg.App.Env)
}

// initStorage создаёт клиенты хранилищ в фиксированном порядке:
// PostgreSQL → Redis → NATS. Порядок создания задаёт порядок регистрации
// в closer, а значит — обратный порядок закрытия: NATS → Redis → PostgreSQL.
func (a *App) initStorage(ctx context.Context) {
	a.diContainer.PGPool(ctx)
	a.diContainer.RedisClient()
	a.diContainer.NATSClient()
}

// initListener открывает TCP-порт HTTP-сервера. Занятый порт — повод упасть сразу.
func (a *App) initListener(ctx context.Context) {
	listenConfig := net.ListenConfig{}

	listener, err := listenConfig.Listen(ctx, "tcp", a.cfg.HTTP.Addr)
	if err != nil {
		slog.Error("открыть порт HTTP-сервера", "address", a.cfg.HTTP.Addr, "error", err)
		os.Exit(1)
	}

	a.listener = listener
}

// initHTTPServer собирает HTTP-сервер и регистрирует его остановку в closer.
//
// Хранилища уже зарегистрированы (initStorage), сервер — после них:
// при LIFO-закрытии он остановится первым, и текущие запросы доработают
// с ещё живыми пулом, Redis и NATS.
func (a *App) initHTTPServer(ctx context.Context) {
	a.httpServer = &http.Server{
		Handler:      newHTTPHandler(a.cfg, a.diContainer.Routes(ctx)),
		ReadTimeout:  a.cfg.HTTP.ReadTimeout,
		WriteTimeout: a.cfg.HTTP.WriteTimeout,
		IdleTimeout:  a.cfg.HTTP.IdleTimeout,
	}

	closer.Add("HTTP-сервер", a.stopHTTPServer)
}

// runHTTPServer блокирует, пока сервер работает. После Shutdown метод Serve
// возвращает http.ErrServerClosed — это штатное завершение, а не ошибка.
func (a *App) runHTTPServer() error {
	slog.Info("HTTP-сервер NetQuest API запущен", "address", a.listener.Addr().String())

	if err := a.httpServer.Serve(a.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("работа HTTP-сервера: %w", err)
	}

	return nil
}

// stopHTTPServer — closer-функция HTTP-сервера.
//
// Shutdown закрывает listener и ждёт, пока текущие запросы доработают.
// Если срок вышел, Shutdown возвращает ошибку, но оставшиеся соединения
// не рвёт — это делает Close.
func (a *App) stopHTTPServer(ctx context.Context) error {
	if err := a.httpServer.Shutdown(ctx); err != nil {
		if closeErr := a.httpServer.Close(); closeErr != nil {
			slog.Error("принудительно закрыть HTTP-сервер", "error", closeErr)
		}

		return fmt.Errorf("остановить HTTP-сервер в срок: %w", err)
	}

	return nil
}
