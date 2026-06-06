#!/usr/bin/env sh
set -eu

command -v docker >/dev/null 2>&1 || { echo "docker is missing"; exit 1; }
docker compose version >/dev/null
df -h /
ss -tulpn | grep -E ':80|:443' || true
echo "Remote checks completed"
