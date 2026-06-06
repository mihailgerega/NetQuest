#!/usr/bin/env sh
set -eu

if [ "${1:-}" = "" ]; then
  echo "Usage: restore-postgres.sh backups/netquest-YYYYmmddTHHMMSSZ.sql" >&2
  exit 1
fi

APP_DIR="${APP_DIR:-/opt/netquest}"
cd "$APP_DIR"
docker compose --env-file .env.production -f docker-compose.prod.yml exec -T postgres sh -c 'psql -U "$POSTGRES_USER" "$POSTGRES_DB"' < "$1"
