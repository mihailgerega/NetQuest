# Архитектура NetQuest

NetQuest состоит из frontend, backend, PostgreSQL и вспомогательной инфраструктуры для локального и production-запуска.

## Frontend

Frontend построен на Next.js, React, TypeScript и Tailwind CSS.

Основные экраны:

- landing page;
- login/register/demo login;
- dashboard проектов;
- simulator;
- Quest Mode;
- documentation.

Simulator хранит локальное состояние рабочего поля, показывает palette, инспектор, Timeline, инспектор пакета, протокольный разбор и Validation Advisor. Перед simulation он сохраняет текущую topology и запускает backend только по свежему `topologyId`.

## Backend

Backend написан на Go и разложен по слоям: запрос идёт сверху вниз, а зависимости
объявлены интерфейсами у потребителя (`deps.go` в каждом пакете).

```text
cmd/api, cmd/migrate        точки входа: конфиг → app; CLI миграций
internal/app                DI-контейнер (di.go), маршруты (router.go), жизненный цикл и graceful shutdown
internal/middleware         RequestID, Recover, security headers, CORS, лимит тела, таймаут, лог, RequireAuth
internal/api/<домен>/v1     HTTP-хендлеры: разбор запроса → сервис → ответ
internal/contract/<домен>/v1  JSON-контракт запросов и ответов API
internal/api/converter      контракт ↔ input сервиса и модели
internal/service/application  сервисы: auth, project, topology, simulation, quest, advisor, audit, health
internal/service/domain     доменная логика без ввода-вывода: validator, engine (Simulation Engine),
                            checker (проверка квестов), catalog (каталог упражнений)
internal/repository/<домен> SQL поверх pgx; record — строки таблиц, converter — строки → модели
internal/producer           публикация событий симуляции в NATS
internal/model              доменные модели
internal/errors (errs)      доменные ошибки; в HTTP-коды их переводит только httpx.WriteError
internal/httpx, ratelimit, metrics, security, config
pkg/closer, logger, nats, migrator, postgres, idgen   переиспользуемая инфраструктура
```

Поток запроса на примере запуска симуляции:

```text
HTTP POST /api/v1/simulations
→ middleware (RequestID … RateLimit, Metrics) → RequireAuth (JWT → Principal в ctx)
→ api/simulation/v1.Start: JSON → simulationv1.StartRequest → input.StartSimulationInput
→ service/application/simulation.Start: проверка проекта и версии → строка simulations
→ domain/engine.Run (валидация, события, сводка) → repository: события и итог → NATS
→ converter → simulationv1.StartResponse → 201
```

Тесты лежат в подкаталогах `tests/` рядом с пакетами и используют testify; моки
интерфейсов из `deps.go` генерирует mockery (`.mockery.yaml`) в подкаталоги `mocks/`.
Стиль кода проверяет golangci-lint (`backend/.golangci.yml`).

## Данные

PostgreSQL хранит пользователей, refresh tokens, проекты, версии topology, simulations, события, quest attempts и прогресс подсказок.

Topology сохраняется как JSON-документ. Это позволяет быстро развивать модель узлов и links без тяжёлой миграции под каждое поле.

## Поток simulation

1. Пользователь меняет canvas.
2. Frontend помечает проект как unsaved.
3. При Run frontend выполняет autosave.
4. Backend создаёт новую topology version.
5. Frontend запускает simulation по свежему `topologyId`.
6. Backend загружает topology из базы, нормализует и валидирует её.
7. Simulation Engine рассчитывает события и summary.
8. Frontend показывает Timeline, inspectors и подсветку path.

## Безопасность

Refresh token хранится в HttpOnly cookie. Backend отдаёт security headers, а production-сценарий использует Caddy для HTTPS. NetQuest не отправляет настоящие network packets: все DNS/Ping/HTTPS-события виртуальные.
