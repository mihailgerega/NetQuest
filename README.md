# NetQuest

NetQuest — учебный симулятор компьютерных сетей. В нём можно собрать топологию из клиентов, серверов, DNS, маршрутизаторов, firewall, балансировщиков и каналов связи, запустить виртуальный DNS Lookup, Ping или HTTPS-запрос и увидеть, почему пакет дошёл, задержался или завершился ошибкой.

Симулятор не отправляет настоящие сетевые пакеты. Все события рассчитываются на серверной части по сохранённой JSON-топологии.

## Что уже умеет проект

- локальная регистрация, вход, демо-аккаунт и refresh token в HttpOnly cookie;
- проекты и сохранение версий топологий;
- визуальный canvas с узлами, каналами связи, масштабом, перемещением и центрированием;
- запуск DNS, Ping, HTTPS и failover-сценариев;
- виртуальное время симуляции на основе задержек, потерь пакетов, пути и обработки на узлах;
- Timeline событий, инспектор пакета, протокольный разбор и Validation Advisor;
- управляемый пул серверов у Load Balancer;
- Quest Mode с упражнениями, прогрессом, подсказками и backend-проверкой решения;
- Docker Compose для локального запуска и production-compose с Caddy.

## Быстрый сценарий

1. Откройте страницу входа.
2. Создайте локальный аккаунт или нажмите «Открыть демо».
3. Откройте dashboard и создайте проект.
4. Нажмите «Загрузить demo topology».
5. Выберите источник запроса, домен, URL и seed.
6. Запустите DNS, Ping или HTTPS.
7. Изучите Timeline, инспектор пакета и протокольный разбор.
8. Измените DNS, маршрут, firewall, канал связи или пул серверов.
9. Запустите симуляцию снова и сравните результат.

## Стек

- Backend: Go, PostgreSQL, WebSocket, миграции SQL.
- Frontend: Next.js, React, TypeScript, Tailwind CSS.
- Инфраструктура: Docker Compose, Caddy для production-сценария.

## Локальный запуск

Скопируйте пример переменных окружения:

```powershell
сp .env.example .env
```

Запустите сервисы:

```powershell
docker compose up --build
```

Если Docker падает на `failed to fetch anonymous token` или нужно временно проверить всё через `localhost`, используйте [docs/local-development.md](docs/local-development.md).

Откройте:

- frontend: `http://localhost:3000`
- backend health: `http://localhost:8080/health/ready`
- metrics: `http://localhost:8080/metrics`

## Auth и demo login

NetQuest поддерживает два удобных пути:

- локальный аккаунт: пользователь создаётся в вашей базе, проекты и прогресс упражнений привязаны к нему;
- demo login: быстрый вход для просмотра приложения без ручного создания пользователя.

Access token хранится на frontend, а refresh token выдаётся в HttpOnly cookie `netquest_refresh_token`. Это снижает риск утечки refresh token через JavaScript.

## API

Основные группы endpoint:

- `/api/v1/auth/*` — регистрация, вход, демо-вход, refresh, logout;
- `/api/v1/projects/*` — проекты и сохранённые топологии;
- `/api/v1/simulations` — запуск симуляции по сохранённой topology version;
- `/api/v1/simulations/{id}/events` — события Timeline;
- `/api/v1/advisor/validate` — подсказки по ошибкам топологии;
- `/api/v1/quests/*` — каталог упражнений, попытки, подсказки и проверка решения.

## Simulation Timing and Failover Behavior

`timestampMs` и `summary.totalLatencyMs` — это виртуальное время симуляции. Оно не измеряет настоящую сеть.

Внутри engine работает виртуальный clock:

- topology validation добавляет небольшую задержку проверки;
- DNS lookup учитывает путь до DNS-узла туда и обратно;
- route lookup, firewall decision и load balancer decision добавляют задержку обработки;
- Ping использует приближённый RTT: путь туда и обратно плюс обработка;
- HTTPS добавляет DNS, TCP-модель, TLS-модель, firewall, выбор сервера и доставку до сервера приложения;
- `latencyMs` каждого активного канала на выбранном пути увеличивает итоговое время;
- `packetLossPercent` учитывается детерминированно по seed;
- failover добавляет overhead и показывает, какие серверы были исключены.

Load Balancer выбирает сервер только из актуального пула. Сервер считается доступным, если он существует, имеет тип `server`, включён в пул, не выключен, имеет подходящий порт и reachable path через активные links. Если все серверы недоступны, симуляция завершается понятной ошибкой.

Server nodes теперь могут показывать открытые порты через `config.openPorts[]`. Для HTTPS backend проверяет `tcp/443`: если route и Firewall настроены правильно, но выбранный Server не слушает этот порт, Timeline показывает `server.port.closed`, а Packet/Protocol Inspector объясняет причину.

Перед запуском frontend автоматически сохраняет текущую топологию. Это защищает от stale topology: backend симулирует именно свежую сохранённую версию.

## Quest Mode

Quest Mode — режим упражнений. Каждое упражнение загружает сломанную топологию, формулирует цель и проверяет решение на backend.

Подсказки раскрываются постепенно:

1. сначала объясняется симптом;
2. затем указывается слой сети;
3. дальше подсказывается конкретный элемент;
4. последняя подсказка почти доводит до исправления.

После успешной проверки пользователь видит объяснение решения, словарь терминов и практическую причину, почему такой сбой важен в реальных сетях.

## Документация

- [Архитектура](docs/architecture.md)
- [Элементы сети](docs/network-elements.md)
- [Simulation Engine](docs/simulation-engine.md)
- [Quest Mode](docs/quests.md)
- [Security](docs/security.md)
- [Deployment](docs/deployment.md)

## Проверки

Backend:

```powershell
cd backend
go test ./...
go vet ./...
go build ./cmd/api
go build ./cmd/migrate
```

Frontend:

```powershell
cd frontend
npm run typecheck
npm.cmd run lint
npm run test
npm run build
```

Docker:

```powershell
docker compose config --quiet
docker compose -f docker-compose.prod.yml config --quiet
```

Frontend `npm run test` запускает лёгкие smoke-tests без дополнительных зависимостей. Они проверяют ключевые acceptance-признаки: landing sections, Quest filters/progress, canvas viewport persistence и отсутствие старого hardcoded break behavior.

## Production deployment

Production-compose поднимает frontend, backend, PostgreSQL, миграции и Caddy. Перед запуском заполните переменные:

- `DOMAIN`
- `ACME_EMAIL`
- `PUBLIC_APP_URL`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DB`
- `JWT_SECRET`
- `REFRESH_SECRET`

Проверить конфигурацию:

```powershell
docker compose -f docker-compose.prod.yml config --quiet
```

## Что проверять вручную перед сдачей

1. Увеличить `latencyMs` на link и убедиться, что `totalLatencyMs` вырос.
2. Добавить `Server-3`, подключить его к Load Balancer и добавить в пул.
3. Сломать `Server-1` через Break Selected Node и убедиться, что Load Balancer выбрал другой сервер.
4. Сломать link к серверу и проверить альтернативный path или понятную ошибку.
5. Изменить canvas и сразу нажать Run HTTPS: frontend должен сначала сохранить свежую topology version.
6. Открыть Quest Mode, раскрыть подсказки, исправить топологию и нажать «Проверить решение».

## Текущий статус

Проект находится в demo/MVP-стадии, но основные end-to-end сценарии уже реализованы: frontend → backend → database → simulation → timeline → quest checker.
# network
