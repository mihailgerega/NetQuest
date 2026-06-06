#!/usr/bin/env sh
set -eu

APP_DIR="${APP_DIR:-/opt/netquest}"
cd "$APP_DIR"
mkdir -p backups
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
docker compose --env-file .env.production -f docker-compose.prod.yml exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB"' > "backups/netquest-${timestamp}.sql"
echo "Backup written to backups/netquest-${timestamp}.sql"
