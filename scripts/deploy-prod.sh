#!/usr/bin/env bash
# =====================================================================
# 翻譯員打卡系統 — 正式環境一鍵部署腳本
# ---------------------------------------------------------------------
# 功能：
#   1. 檢查 Docker 是否就緒
#   2. 沒有 backend/.env.production 就從範本複製一份
#   3. 自動產生缺少的 JWT_SECRET / DB_PASSWORD / ADMIN_DEFAULT_PASSWORD
#   4. 驗證必要設定（弱密碼會擋下來）
#   5. 建置並啟動正式環境容器
#   6. 等待服務就緒，最後印出登入資訊
#
# 用法：
#   scripts/deploy-prod.sh             # 部署（不含對外 Tunnel）
#   scripts/deploy-prod.sh --tunnel    # 部署並啟動 Cloudflare Tunnel 對外公開
#   scripts/deploy-prod.sh --help      # 顯示說明
#
# 此腳本可重複執行（升級 / 改設定後再跑一次即可），資料不會遺失。
# =====================================================================
set -euo pipefail

# ---- 顏色（讓重點訊息好讀）-----------------------------------------
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()  { echo -e "${BLUE}➜${NC}  $*"; }
ok()    { echo -e "${GREEN}✓${NC}  $*"; }
warn()  { echo -e "${YELLOW}⚠${NC}  $*"; }
err()   { echo -e "${RED}✗${NC}  $*" >&2; }
die()   { err "$*"; exit 1; }

# ---- 路徑：不管在哪裡執行，都定位到專案根目錄 ----------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

ENV_FILE="backend/.env.production"
ENV_EXAMPLE="backend/.env.production.example"
COMPOSE_FILE="docker/docker-compose.prod.yml"

# ---- 解析參數 -------------------------------------------------------
USE_TUNNEL=0
for arg in "$@"; do
  case "$arg" in
    --tunnel) USE_TUNNEL=1 ;;
    -h|--help)
      grep '^#' "$0" | sed 's/^# \{0,1\}//'
      exit 0 ;;
    *) die "未知參數：$arg（用 --help 看說明）" ;;
  esac
done

# ---------------------------------------------------------------------
# 小工具：讀 / 寫 .env 檔裡的某個 KEY
# ---------------------------------------------------------------------
get_env() { grep -E "^$1=" "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2- || true; }

set_env() { # set_env KEY VALUE  —— 有就替換，沒有就新增（不破壞其他行）
  local key="$1" val="$2" tmp
  tmp="$(mktemp)"
  awk -v k="$key" -v v="$val" '
    $0 ~ "^"k"=" { print k"="v; done=1; next }
    { print }
    END { if (!done) print k"="v }
  ' "$ENV_FILE" > "$tmp" && mv "$tmp" "$ENV_FILE"
}

# =====================================================================
# 1. 檢查 Docker
# =====================================================================
info "檢查 Docker 環境…"
command -v docker >/dev/null 2>&1 || die "找不到 docker，請先安裝 Docker。"
docker compose version >/dev/null 2>&1 || die "找不到 'docker compose'（v2），請更新 Docker。"
docker info >/dev/null 2>&1 || die "Docker daemon 沒有在執行，請先啟動 Docker。"
ok "Docker 就緒"

# =====================================================================
# 2. 準備 .env.production
# =====================================================================
if [[ ! -f "$ENV_FILE" ]]; then
  [[ -f "$ENV_EXAMPLE" ]] || die "找不到範本 $ENV_EXAMPLE"
  cp "$ENV_EXAMPLE" "$ENV_FILE"
  warn "沒有 $ENV_FILE，已從範本複製一份。稍後會自動補上必要密鑰。"
  warn "選填整合（SMTP / Google / LINE / Tunnel）請之後自行編輯 $ENV_FILE 再重跑。"
fi

# =====================================================================
# 3. 自動產生缺少的密鑰（已存在的不動）
# =====================================================================
info "檢查並補齊安全密鑰…"
GENERATED=0

# JWT_SECRET：需 >=32 字且非預設值
JWT="$(get_env JWT_SECRET)"
if [[ -z "$JWT" || "$JWT" == "dev-secret-key-change-in-production" || ${#JWT} -lt 32 ]]; then
  JWT="$(openssl rand -hex 32)"
  set_env JWT_SECRET "$JWT"
  ok "已自動產生 JWT_SECRET"
  GENERATED=1
fi

# DB_PASSWORD：不可空、不可是預設 postgres
DBPW="$(get_env DB_PASSWORD)"
if [[ -z "$DBPW" || "$DBPW" == "postgres" ]]; then
  DBPW="$(openssl rand -base64 24 | tr -d '\n/+=' | cut -c1-24)"
  set_env DB_PASSWORD "$DBPW"
  ok "已自動產生 DB_PASSWORD"
  GENERATED=1
fi

# ADMIN_DEFAULT_PASSWORD：只在第一次建立 admin 時生效，這裡確保有值且可印出
ADMINPW="$(get_env ADMIN_DEFAULT_PASSWORD)"
if [[ -z "$ADMINPW" ]]; then
  ADMINPW="$(openssl rand -base64 18 | tr -d '\n/+=' | cut -c1-16)"
  set_env ADMIN_DEFAULT_PASSWORD "$ADMINPW"
  ok "已自動產生 ADMIN_DEFAULT_PASSWORD"
  GENERATED=1
fi

[[ "$GENERATED" == 1 ]] && warn "已寫入新密鑰到 $ENV_FILE，請妥善保管、勿外流。"

# =====================================================================
# 4. 驗證設定
# =====================================================================
info "驗證設定…"
JWT="$(get_env JWT_SECRET)";   [[ ${#JWT} -ge 32 ]] || die "JWT_SECRET 長度不足 32 字。"
DBPW="$(get_env DB_PASSWORD)"; [[ -n "$DBPW" && "$DBPW" != "postgres" ]] || die "DB_PASSWORD 無效或仍為預設值。"

if [[ "$USE_TUNNEL" == 1 ]]; then
  TOKEN="$(get_env TUNNEL_TOKEN)"
  [[ -n "$TOKEN" ]] || die "你加了 --tunnel，但 $ENV_FILE 裡的 TUNNEL_TOKEN 是空的。請先到 Cloudflare 建立 Tunnel 取得 token。"
  ok "Cloudflare Tunnel 設定就緒"
fi
ok "設定驗證通過"

# =====================================================================
# 5. 建置並啟動
# =====================================================================
COMPOSE=(docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE")
[[ "$USE_TUNNEL" == 1 ]] && COMPOSE+=(--profile tunnel)

info "建置並啟動容器（第一次會比較久，請耐心等）…"
"${COMPOSE[@]}" up -d --build
ok "容器已啟動"

# =====================================================================
# 6. 等待服務就緒
# =====================================================================
info "等待服務就緒（最多 120 秒）…"
ready=0
for _ in $(seq 1 60); do
  # 前端回 200、且透過前端打後端 /api 有回應（缺 body 會回 400），代表整條鏈路通了
  front="$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:3000/ 2>/dev/null || echo 000)"
  back="$(curl -s -o /dev/null -w '%{http_code}' -X POST http://127.0.0.1:3000/api/auth/login 2>/dev/null || echo 000)"
  if [[ "$front" == "200" && "$back" =~ ^(400|401|422)$ ]]; then
    ready=1; break
  fi
  sleep 2
done

echo
"${COMPOSE[@]}" ps
echo

if [[ "$ready" == 1 ]]; then
  ok "服務已就緒 🎉"
else
  warn "等待逾時，服務可能還在啟動或有錯誤。請看後端 log："
  echo "    ${COMPOSE[*]} logs -f backend"
fi

# =====================================================================
# 登入資訊
# =====================================================================
echo
echo "==================== 登入資訊 ===================="
echo "  管理員帳號： admin@admin.com"
echo "  初始密碼  ： $(get_env ADMIN_DEFAULT_PASSWORD)"
echo "  （首次登入後系統會強制你改密碼）"
echo "=================================================="
echo
if [[ "$USE_TUNNEL" == 1 ]]; then
  info "對外網址：由你在 Cloudflare Tunnel 的 Public Hostname 設定的網域。"
else
  info "目前僅本機可連：http://127.0.0.1:3000"
  info "要對外公開給外出的翻譯員，請設定 Cloudflare Tunnel 後用 --tunnel 重跑。"
  info "詳見 docs/PRODUCTION_DEPLOY.md"
fi
echo
info "常用指令："
echo "    查看狀態： ${COMPOSE[*]} ps"
echo "    看後端 log： ${COMPOSE[*]} logs -f backend"
echo "    停止服務： ${COMPOSE[*]} down        （資料保留）"
echo "    重設密碼： docker compose -f $COMPOSE_FILE exec backend ./server -reset-password admin@admin.com 'NewPass123'"
