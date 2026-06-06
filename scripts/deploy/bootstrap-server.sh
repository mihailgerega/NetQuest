#!/usr/bin/env sh
set -eu

sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg git ufw

if ! command -v docker >/dev/null 2>&1; then
  curl -fsSL https://get.docker.com | sudo sh
fi

sudo usermod -aG docker "$USER" || true
sudo mkdir -p /opt/netquest
sudo chown "$USER":"$USER" /opt/netquest
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

echo "Server bootstrap complete. Re-login may be required for docker group access."
