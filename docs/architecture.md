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

Backend написан на Go.

Основные модули:

- `auth` — регистрация, login, demo login, refresh/logout;
- `projects` — проекты и версии topology;
- `topology` — normalization и validation;
- `simulation` — расчёт виртуальных событий, задержки, пути и failover;
- `advisor` — диагностика topology до запуска;
- `quests` — каталог упражнений, попытки, hints и checker;
- `realtime` — stream событий с fallback на polling;
- `security` — базовые security headers.

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
