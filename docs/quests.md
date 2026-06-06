# Quest Mode

Quest Mode — режим упражнений NetQuest. Он нужен, чтобы учиться диагностировать сетевые сбои не по готовому ответу, а через практику: открыть сломанную topology, запустить simulation, понять первый симптом, исправить причину и отправить решение на backend-проверку.

## Как открыть

1. Войдите в локальный аккаунт или demo-account.
2. Нажмите «Упражнения» на landing page, dashboard или в верхней панели simulator.
3. Выберите карточку упражнения.
4. Изучите цель, ожидаемые проверки, словарь и подсказки.
5. Нажмите «Начать упражнение».

## Каталог

Упражнения отсортированы от простых к сложным:

- easy — базовые ошибки DNS, Ping, link status и source client;
- medium — routing, Load Balancer, stale references, default gateway и latency;
- hard — failover, security rules, backup path, TLS и multi-layer incidents.

Каждое упражнение имеет категорию: DNS, Routing, Firewall, Load Balancer, Latency, Failover, Security или TLS. Frontend показывает общий прогресс и прогресс по сложности.

## Подсказки

Подсказки раскрываются постепенно и сохраняются в attempt:

1. первая подсказка объясняет симптом;
2. вторая направляет к слою сети;
3. третья указывает, где искать в инспекторе или Timeline;
4. четвёртая говорит, что изменить и как проверить;
5. hard-упражнения имеют пятую подсказку, которая почти доводит до решения.

Цель подсказок — помочь новичку понять причину, а не просто скопировать ответ.

## Проверка решения

Кнопка «Проверить решение» отправляет текущую topology на backend. Checker запускает simulation и проверяет expected checks:

- DNS-запись и фактический DNS-ответ;
- reachable path и статус simulation;
- firewall decision;
- состав пула Load Balancer;
- выбранный и пропущенные серверы;
- задержку и SLA;
- отсутствие stale references.

Если решение не прошло, пользователь видит failed checks и следующую полезную подсказку.

## После успешного решения

NetQuest показывает:

- почему исправление сработало;
- какие checks подтвердили решение;
- glossary по теме упражнения;
- почему такой сбой встречается в реальной инфраструктуре.

## Как писать новые упражнения

Для нового quest нужны:

- slug, title, category, difficulty и estimatedMinutes;
- broken initialTopology;
- expectedChecks;
- минимум 4 progressive hints, а для hard — минимум 5;
- afterSolution explanation;
- glossary terms;
- realWorldImportance.

Проверка должна быть поведенческой: не только “узел есть на canvas”, а “симуляция проходит ожидаемым маршрутом и принимает правильные решения”.
