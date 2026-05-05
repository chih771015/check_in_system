#!/usr/bin/env bash
# expose.sh — 啟動測試環境並透過 cloudflared 對外開放
#
# 使用方式：
#   bash scripts/expose.sh          # 預設監聽 port 3000
#   bash scripts/expose.sh 3001     # 指定其他 port
#
# 需求：
#   - Docker Desktop（已執行）
#   - macOS + Homebrew（會自動安裝 cloudflared）

set -e

COMPOSE_DIR="$(cd "$(dirname "$0")/../docker" && pwd)"
EXPOSE_PORT="${1:-3000}"

# ── 1. 確認 cloudflared 已安裝 ──────────────────────────────────────────────
if ! command -v cloudflared &>/dev/null; then
  echo "📦  找不到 cloudflared，透過 Homebrew 安裝中..."
  brew install cloudflared
fi
echo "✅  cloudflared $(cloudflared --version 2>&1 | head -1)"

# ── 2. 啟動（或重啟）Docker Compose（含 expose nginx conf）────────────────
echo ""
echo "🐳  啟動 Docker Compose（使用 expose 設定）..."
docker compose \
  -f "$COMPOSE_DIR/docker-compose.yml" \
  -f "$COMPOSE_DIR/docker-compose.expose.yml" \
  up -d --build

echo ""
echo "⏳  等待 backend 就緒..."
until curl -sf "http://localhost:$EXPOSE_PORT/api/health" &>/dev/null || \
      curl -sf "http://localhost:$EXPOSE_PORT/" &>/dev/null; do
  sleep 2
  printf "."
done
echo ""
echo "✅  服務就緒 → http://localhost:$EXPOSE_PORT"

# ── 3. 建立 cloudflared 快速穿透隧道 ────────────────────────────────────────
echo ""
echo "🌐  建立 cloudflared 隧道，請稍候..."
echo "    （按 Ctrl+C 關閉隧道，Docker 服務會繼續在背景執行）"
echo "    （若要同時停止 Docker，改按 Ctrl+C 後執行 docker/stop.sh）"
echo ""

cloudflared tunnel --url "http://localhost:$EXPOSE_PORT"
