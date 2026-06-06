# Simulation Engine

Simulation Engine — серверная часть NetQuest, которая получает сохранённую topology version и рассчитывает виртуальные события сети. Engine не отправляет реальные пакеты и не открывает сетевые соединения.

## Контракт запуска

Frontend перед каждым запуском сохраняет текущую topology. После успешного save он передаёт в `/api/v1/simulations` свежий `topologyId`. Backend загружает эту версию из базы и запускает simulation по ней.

Такой контракт защищает от stale topology: новые узлы, удалённые links, изменённые статусы и latency попадают в расчёт сразу.

## Simulation Timing and Failover Behavior

`timestampMs` в Timeline рассчитывается виртуальным clock:

1. clock начинается с `0`;
2. каждое событие получает текущее значение;
3. сетевые и processing-этапы увеличивают clock;
4. timestamps остаются monotonic;
5. `summary.totalLatencyMs` соответствует итоговому виртуальному времени.

На время влияют:

- `latencyMs` links на выбранном path;
- DNS lookup туда и обратно;
- route lookup;
- firewall decision;
- TCP handshake model;
- TLS handshake model;
- Load Balancer decision;
- доставка до выбранного сервера;
- retry overhead при deterministic packet loss;
- failover overhead.

Потеря пакетов считается по seed. При одинаковой topology и seed результат повторяется.

## DNS Lookup

Engine ищет DNS-узел, проверяет его состояние и A-запись для домена. Время DNS складывается из пути до DNS, задержки обработки и обратного пути. Если запись отсутствует или DNS недоступен, Timeline получает понятную ошибку.

## Routing

Route строится по активным links и состояниям nodes. Если включены routing tables, engine учитывает gateway, route destination, interface и cost. Если route отсутствует, возвращается `route.not_found` с объяснением.

## Firewall

Firewall проверяет трафик по протоколу, порту, источнику и назначению. Первое подходящее правило определяет действие. Если трафик заблокирован, simulation завершается событием с ошибкой и сохраняет частичный путь.

## Load Balancer

Load Balancer выбирает backend только из актуального пула:

- backend node должен существовать;
- node должен иметь type `server`;
- backend должен быть enabled;
- server должен быть `healthy` или допустимо degraded;
- от Load Balancer до server должен существовать active path;
- port сервера должен соответствовать назначению.

Если включено auto-discovery, подключённые server nodes добавляются как candidates. Timeline показывает `lb.backend.discovered`, `lb.backend.selected` или `lb.backend.unhealthy`.

Event выбора содержит:

- выбранный node id и name;
- algorithm;
- reason;
- доступные серверы;
- пропущенные серверы с причинами.

Если доступных серверов нет, simulation завершается ошибкой: `Load balancer has no healthy backends available.`

## Инспекторы

Simulation response содержит:

- id simulation;
- status;
- summary;
- события;
- protocol details;
- разбор задержки;
- decisions;
- errors.

Timeline показывает номер события, время, тип, источник, цель, сообщение, прирост задержки и важность. Инспектор пакета показывает статус, `totalLatencyMs`, путь, DNS-результат, решение firewall, выбранный сервер, пропущенные серверы, failover-события и ошибки.

## Quest Checker

Quest Checker использует тот же engine, что и обычный simulator. Проверка упражнения основана на поведении topology: проходит ли DNS, есть ли route, разрешён ли traffic, выбран ли корректный backend и соблюдается ли latency target.
