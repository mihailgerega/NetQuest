#!/usr/bin/env sh
set -eu

APP_DIR="${APP_DIR:-/opt/netquest}"

cd "$APP_DIR"
if [ ! -f ".env.production" ]; then
  echo "Missing .env.production in $APP_DIR" >&2
  exit 1
fi

git pull --ff-only || true
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
docker compose --env-file .env.production -f docker-compose.prod.yml ps
docker compose --env-file .env.production -f docker-compose.prod.yml exec -T backend /app/migrate up
docker compose --env-file .env.production -f docker-compose.prod.yml exec -T backend wget -qO- http://localhost:8080/health/live
