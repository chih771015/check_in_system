#!/usr/bin/env bash
# stop.sh — 停止 Docker Compose 測試環境

COMPOSE_DIR="$(cd "$(dirname "$0")/../docker" && pwd)"

echo "🛑  停止測試環境..."
docker compose \
  -f "$COMPOSE_DIR/docker-compose.yml" \
  -f "$COMPOSE_DIR/docker-compose.expose.yml" \
  down

echo "✅  所有服務已停止"
