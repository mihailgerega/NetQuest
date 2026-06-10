# NetQuest Deployment

Target host: Ubuntu 22.04/24.04 VPS with Docker and Docker Compose plugin. Production uses Docker Compose and Caddy for automatic HTTPS.

## Server Requirements

- 2 CPU, 2 GB RAM minimum for demo traffic.
- Ports open to the internet: `80/tcp`, `443/tcp`.
- PostgreSQL, Redis and NATS are internal Docker services and must not be published publicly.
- DNS A-record: `DOMAIN -> server public IPv4`.

## Bootstrap

```sh
scp -r . user@server:/opt/netquest
ssh user@server
cd /opt/netquest
./scripts/deploy/bootstrap-server.sh
cp .env.production.example .env.production
```

Edit `.env.production`:

- `DOMAIN`
- `ACME_EMAIL`
- `PUBLIC_APP_URL`
- `NEXT_PUBLIC_API_BASE_URL`
- `NEXT_PUBLIC_REPOSITORY_URL`
- `POSTGRES_PASSWORD`
- `POSTGRES_DSN`
- `JWT_SECRET`
- `CORS_ALLOWED_ORIGINS`
- `SECURE_COOKIE=true`

Use a long random `JWT_SECRET` and a non-default database password.

For the public deployment on `net-quest.ru`, the browser-facing values should be:

```env
DOMAIN=net-quest.ru
PUBLIC_APP_URL=https://net-quest.ru
NEXT_PUBLIC_API_BASE_URL=https://net-quest.ru
CORS_ALLOWED_ORIGINS=https://net-quest.ru
SECURE_COOKIE=true
```

Do not use `docker-compose.yml` on the VPS for production. It is the local development file and can build the browser bundle for `localhost`.

## Run

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
docker compose --env-file .env.production -f docker-compose.prod.yml ps
```

`NEXT_PUBLIC_API_BASE_URL` is baked into the browser bundle during the frontend image build. If `PUBLIC_APP_URL` or repository URL changes, rebuild the frontend image with `--build`; a plain container restart is not enough.

The backend container runs migrations before API startup. You can also force migrations:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml exec -T backend /app/migrate up
```

## HTTPS

Caddy reads [docker/caddy/Caddyfile](docker/caddy/Caddyfile). It proxies:

- `/` to frontend
- `/api/*`, `/health/*`, `/metrics` to backend
- `/api/v1/ws` supports WebSocket upgrade through the same reverse proxy path

Caddy obtains Let's Encrypt certificates automatically when DNS is pointed at the server and ports 80/443 are reachable.

## Checks

```sh
curl -I https://$DOMAIN/
curl https://$DOMAIN/health/live
curl https://$DOMAIN/health/ready
docker compose --env-file .env.production -f docker-compose.prod.yml logs -f backend
```

Deployment checklist:

- frontend opens publicly;
- HTTPS certificate is valid;
- `/health/live` returns 200 through Caddy;
- demo login works;
- auth responses set `netquest_refresh_token` as httpOnly cookie and do not expose refresh token in JSON;
- demo topology loads;
- simulation starts;
- timeline receives events through WebSocket or polling fallback;
- `/docs` opens;
- `/docs` shows GitHub link when `NEXT_PUBLIC_REPOSITORY_URL` is set;
- PostgreSQL/Redis/NATS are not exposed publicly;
- `docker compose ps` shows healthy/running services;
- logs do not contain passwords or tokens.

## Backup

```sh
APP_DIR=/opt/netquest ./scripts/deploy/backup-postgres.sh
```

Backups are written to `backups/netquest-<timestamp>.sql`.

Restore:

```sh
APP_DIR=/opt/netquest ./scripts/deploy/restore-postgres.sh backups/netquest-20260604T120000Z.sql
```

## Update and Rollback

Update:

```sh
cd /opt/netquest
git pull --ff-only
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

Rollback:

```sh
git checkout <previous-commit>
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

## Troubleshooting

- Certificate is not issued: check DNS A-record, ports 80/443 and `ACME_EMAIL`.
- Backend not ready: inspect `docker compose logs backend`, then check Postgres/Redis/NATS health.
- Demo auth disabled: set `DEMO_AUTH_ENABLED=true` only for a public demo and understand anyone can create demo projects.
