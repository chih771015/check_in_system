#!/usr/bin/env bash
# =====================================================================
# 翻譯員打卡系統 — 交接打包腳本  /  Handover packaging script
# ---------------------------------------------------------------------
# 用途：把「乾淨、可直接部署」的原始碼打包成一個壓縮檔交給接手的人。
#   Purpose: produce a clean, deploy-ready source archive to hand over.
#
# 特點 / What it does:
#   * 只打包 git 追蹤中的檔案（自動排除 node_modules、dist、uploads、.git）
#     Only packs git-tracked files (no node_modules / dist / uploads / .git)
#   * 強制移除任何機密檔（.env、.env.production、google 憑證），避免外洩
#     Strips every secret file (.env*, google credentials) to avoid leaks
#   * 產出 dist-handover/翻譯員打卡系統_YYYY-MM-DD.tar.gz
#     Outputs the archive under dist-handover/
#
# 用法 / Usage:
#   bash scripts/package-handover.sh            # tar.gz（預設 / default）
#   bash scripts/package-handover.sh --zip      # 改產出 .zip
# =====================================================================
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info() { echo -e "${BLUE}➜${NC}  $*"; }
ok()   { echo -e "${GREEN}✓${NC}  $*"; }
warn() { echo -e "${YELLOW}⚠${NC}  $*"; }
die()  { echo -e "${RED}✗${NC}  $*" >&2; exit 1; }

# ---- 定位專案根目錄 / locate project root --------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

USE_ZIP=0
for arg in "$@"; do
  case "$arg" in
    --zip) USE_ZIP=1 ;;
    -h|--help) grep '^#' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) die "未知參數 / unknown arg：$arg" ;;
  esac
done

command -v git >/dev/null 2>&1 || die "找不到 git / git not found"
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || die "這裡不是 git 專案 / not a git repo"

# ---- 機密檔清單：即使被 git 追蹤也一定要從壓縮檔剔除 ----------------
# Secret files that must never be in the handover, even if git-tracked.
SECRETS=(
  "backend/.env"
  "backend/.env.production"
  ".env"
  ".env.production"
  "backend/google-credentials.json"
)

STAMP="$(date +%Y-%m-%d)"
OUT_DIR="$ROOT_DIR/dist-handover"
STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
mkdir -p "$OUT_DIR"

# ---- 1. 匯出乾淨快照 / export a clean snapshot ---------------------
# 納入「已追蹤」+「未 commit 但未被 gitignore 忽略」的檔案，
# 因此新加、尚未 commit 的檔（如 HANDOVER.md）也會被打包；
# 同時自動排除 node_modules / dist / uploads / .git（都在 .gitignore）。
# Includes tracked files AND new-but-not-ignored files, so freshly added
# files are packed without a commit; .gitignore'd junk is still excluded.
info "匯出乾淨原始碼快照…  /  Exporting clean source snapshot…"
{ git ls-files -z; git ls-files -z --others --exclude-standard; } \
  | ( cd "$ROOT_DIR" && rsync -a0 --files-from=- ./ "$STAGE/" )
ok "已匯出（不含 node_modules / dist / uploads / .git）"

# ---- 2. 剔除機密檔 / strip secrets ---------------------------------
info "剔除機密檔…  /  Stripping secret files…"
for f in "${SECRETS[@]}"; do
  if [[ -e "$STAGE/$f" ]]; then
    rm -f "$STAGE/$f"
    warn "已移除 / removed：$f"
  fi
done

# 保險：掃描壓縮前的內容，若還殘留任何 .env（非 .example）就中止
# Safety net: abort if any non-example .env slips through.
LEFTOVER="$(cd "$STAGE" && find . -name '.env' -o -name '.env.production' -o -name 'google-credentials.json' | grep -v '\.example$' || true)"
[[ -z "$LEFTOVER" ]] || die "仍有機密檔殘留，已中止 / secrets still present:\n$LEFTOVER"
ok "確認無機密檔外洩 / no secrets in archive"

# ---- 3. 打包 / archive --------------------------------------------
BASENAME="翻譯員打卡系統_${STAMP}"
if [[ "$USE_ZIP" == 1 ]]; then
  command -v zip >/dev/null 2>&1 || die "找不到 zip 指令 / zip not installed"
  OUT="$OUT_DIR/${BASENAME}.zip"
  rm -f "$OUT"
  ( cd "$STAGE" && zip -rq "$OUT" . )
else
  OUT="$OUT_DIR/${BASENAME}.tar.gz"
  rm -f "$OUT"
  tar -czf "$OUT" -C "$STAGE" .
fi

SIZE="$(du -h "$OUT" | cut -f1)"
COUNT="$(cd "$STAGE" && find . -type f | wc -l | tr -d ' ')"

echo
ok "打包完成 / Done"
echo "  檔案 / file ：$OUT"
echo "  大小 / size ：$SIZE（$COUNT 個檔案 / files）"
echo
info "交給接手的人後，請他 / Tell the recipient to:"
echo "    1. 解壓縮 / unpack the archive"
echo "    2. 讀 HANDOVER.md（總入口 / start here）"
echo "    3. 跑 bash scripts/deploy-prod.sh（一鍵部署 / one-command deploy）"
