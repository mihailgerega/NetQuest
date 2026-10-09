// Package closer — graceful shutdown: реестр функций закрытия ресурсов
// (пул PostgreSQL, клиент Redis, соединение с NATS, HTTP-сервер), которые
// при остановке вызываются в обратном порядке регистрации.
//
// DI-контейнер регистрирует ресурс в момент создания, app вызывает CloseAll
// после сигнала SIGTERM. Порядок создания задаёт порядок закрытия, поэтому
// в di.go геттеры зависимостей вызываются раньше геттеров тех, кто ими пользуется.
package closer

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// closeFn описывает одну функцию закрытия с именем ресурса.
type closeFn struct {
	name string
	fn   func(context.Context) error
}

// closer управляет процессом graceful shutdown приложения.
// Функции закрытия вызываются в обратном порядке (LIFO) — последний добавленный
// ресурс закрывается первым, что гарантирует корректный порядок зависимостей.
type closer struct {
	mu    sync.Mutex
	once  sync.Once
	funcs []closeFn
}

// globalCloser — глобальный экземпляр, инициализируется при загрузке пакета.
// Благодаря этому closer готов к использованию сразу — его не нужно создавать
// вручную, достаточно вызывать пакетные функции Add и CloseAll.
var globalCloser = newCloser()

// newCloser создаёт новый экземпляр closer.
func newCloser() *closer {
	return &closer{}
}

// Add добавляет функцию закрытия в глобальный closer.
func Add(name string, f func(context.Context) error) {
	globalCloser.Add(name, f)
}

// CloseAll вызывает все функции закрытия глобального closer'а.
func CloseAll(ctx context.Context) error {
	return globalCloser.CloseAll(ctx)
}

// Add добавляет функцию закрытия с именем ресурса.
func (c *closer) Add(name string, f func(context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.funcs = append(c.funcs, closeFn{name: name, fn: f})
}

// CloseAll вызывает все зарегистрированные функции закрытия в обратном порядке (LIFO).
// Безопасен для повторного вызова — выполнится только один раз; повторный вызов
// вернёт nil, даже если первый вернул ошибку.
//
// ctx — общий срок на все функции: если первая съела весь таймаут, следующие
// получат уже истёкший контекст. Ошибка одной функции не останавливает остальные:
// наружу уходит первая ошибка, все — в логе.
func (c *closer) CloseAll(ctx context.Context) error {
	var result error

	c.once.Do(func() {
		c.mu.Lock()
		funcs := c.funcs
		c.funcs = nil
		c.mu.Unlock()

		if len(funcs) == 0 {
			return
		}

		slog.Info("начинаем graceful shutdown", "count", len(funcs))

		// Обходим в обратном порядке (LIFO): ресурсы, добавленные последними, закрываются первыми.
		// Зависимости регистрируются в порядке создания: сначала PostgreSQL, Redis и NATS,
		// потом HTTP-сервер. При завершении сначала останавливаем сервер (перестаём принимать
		// запросы и ждём текущие), и только потом закрываем хранилища, которыми он пользуется.
		for i := len(funcs) - 1; i >= 0; i-- {
			f := funcs[i]

			start := time.Now()
			slog.Info("закрываем ресурс", "name", f.name)

			if err := f.fn(ctx); err != nil {
				slog.Error("ошибка при закрытии ресурса", "name", f.name, "error", err, "duration", time.Since(start))

				if result == nil {
					result = err
				}
			} else {
				slog.Info("ресурс закрыт", "name", f.name, "duration", time.Since(start))
			}
		}

		slog.Info("graceful shutdown завершён", "count", len(funcs))
	})

	return result
}
