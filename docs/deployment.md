# Deployment

NetQuest можно запустить локально через Docker Compose или подготовить production-сценарий с Caddy.

## Локальный запуск

1. Создайте `.env` из примера:

```powershell
Copy-Item .env.example .env
```

2. Запустите контейнеры:

```powershell
docker compose up --build
```

3. Откройте:

- frontend: `http://localhost:3000`;
- backend health: `http://localhost:8080/health/ready`;
- metrics: `http://localhost:8080/metrics`.

## Production variables

Для production-compose заполните:

- `DOMAIN`;
- `ACME_EMAIL`;
- `PUBLIC_APP_URL`;
- `POSTGRES_USER`;
- `POSTGRES_PASSWORD`;
- `POSTGRES_DB`;
- `JWT_SECRET`;
- `REFRESH_SECRET`.

## Проверка compose

```powershell
docker compose config --quiet
docker compose -f docker-compose.prod.yml config --quiet
```

Если production variables не заданы, Docker Compose покажет warnings. Это нормально для локальной проверки структуры, но не подходит для настоящего запуска.

## Production запуск

```powershell
docker compose -f docker-compose.prod.yml up --build -d
```

Caddy получает HTTPS-сертификат для `DOMAIN`, проксирует frontend, backend API и WebSocket stream.

## Smoke checks

После запуска проверьте:

```powershell
curl.exe -s http://localhost:8080/health/live
curl.exe -s http://localhost:8080/health/ready
curl.exe -s http://localhost:8080/metrics
```

Для production используйте публичный domain и endpoint `/api/v1/health/ready`, если reverse proxy настроен именно через API-prefix.
