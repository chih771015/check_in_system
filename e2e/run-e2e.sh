#!/usr/bin/env bash
#
# run-e2e.sh — one-shot E2E runner.
#
# What it does, in order:
#   1. Check prerequisites (docker, node, npm)
#   2. Install npm deps if missing
#   3. Install the Chromium browser for Playwright if missing
#   4. Bring up docker-compose.e2e.yml (postgres + backend with -tags e2e + frontend)
#   5. Wait until the backend's /api/test/reset endpoint answers (max 90s)
#   6. Run Playwright
#   7. Print the report path
#
# Flags:
#   --no-stack        Skip step 4-5 (stack already running)
#   --down            After tests, tear down the stack + delete volumes
#   --ui              Open Playwright UI mode instead of headless run
#   --keep-going      Don't exit on test failures (still tear down if --down)
#   -h, --help        Show this help
#
# Exit codes:
#   0  all green
#   1  prerequisite missing
#   2  stack failed to come up
#   3  tests failed

set -euo pipefail

# ---- config (override via env) ----
COMPOSE_FILE="${COMPOSE_FILE:-../docker/docker-compose.e2e.yml}"
COMPOSE_PROJECT="${COMPOSE_PROJECT:-thai-e2e}"
BASE_URL="${E2E_BASE_URL:-http://localhost:3001}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-90}"

# ---- arg parsing ----
SKIP_STACK=0
TEAR_DOWN=0
UI_MODE=0
KEEP_GOING=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-stack)   SKIP_STACK=1 ;;
    --down)       TEAR_DOWN=1 ;;
    --ui)         UI_MODE=1 ;;
    --keep-going) KEEP_GOING=1 ;;
    -h|--help)
      sed -n '2,30p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) echo "Unknown flag: $1" >&2; exit 1 ;;
  esac
  shift
done

# ---- pretty output helpers ----
RED=$'\033[0;31m'; GRN=$'\033[0;32m'; YEL=$'\033[1;33m'; BLU=$'\033[0;34m'; NC=$'\033[0m'
step()  { echo "${BLU}==>${NC} $*"; }
ok()    { echo "${GRN}✓${NC} $*"; }
warn()  { echo "${YEL}!${NC} $*"; }
fail()  { echo "${RED}✗${NC} $*" >&2; }

# ---- 0. cd into e2e/ regardless of where the script was invoked ----
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ---- 1. prerequisites ----
step "Checking prerequisites"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "$1 not found. Install it and retry."
    exit 1
  fi
}
need_cmd docker
need_cmd node
need_cmd npm
need_cmd curl

# docker-compose v1 OR docker compose v2
if docker compose version >/dev/null 2>&1; then
  DC="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  DC="docker-compose"
else
  fail "Neither 'docker compose' (v2) nor 'docker-compose' (v1) found."
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  fail "Docker daemon is not running. Start Docker Desktop / dockerd and retry."
  exit 1
fi

NODE_MAJOR="$(node -v | sed 's/v\([0-9]*\)\..*/\1/')"
if (( NODE_MAJOR < 18 )); then
  fail "Node $NODE_MAJOR found; need >= 18 for Playwright."
  exit 1
fi
ok "docker / node $NODE_MAJOR / npm / curl available  (compose: $DC)"

# ---- 2. npm install if needed ----
if [[ ! -d node_modules ]]; then
  step "Installing npm dependencies (first run)"
  npm install --no-audit --no-fund
  ok "npm deps installed"
else
  ok "npm deps already installed"
fi

# ---- 3. playwright browsers ----
# The browser cache lives under ~/.cache/ms-playwright (Linux) or
# ~/Library/Caches/ms-playwright (macOS). Easiest portable check is to
# ask Playwright itself.
step "Ensuring Chromium browser is available"
if ! npx playwright install --dry-run chromium 2>/dev/null | grep -q "is already installed"; then
  npx playwright install chromium
fi
ok "Chromium ready"

# ---- 4. stack up ----
if (( SKIP_STACK == 0 )); then
  step "Bringing up E2E stack ($COMPOSE_PROJECT)"
  $DC -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT" up -d --build
  ok "Stack started"

  # ---- 5. wait for backend reset endpoint ----
  step "Waiting for backend to be ready (timeout ${WAIT_TIMEOUT}s)"
  deadline=$(( $(date +%s) + WAIT_TIMEOUT ))
  ready=0
  while (( $(date +%s) < deadline )); do
    # We use POST so we both probe and reset in one shot. 200 = ready.
    if curl -fsS -X POST "$BASE_URL/api/test/reset" -o /dev/null 2>/dev/null; then
      ready=1
      break
    fi
    sleep 2
    printf "."
  done
  echo
  if (( ready == 0 )); then
    fail "Backend never came up. Recent backend logs:"
    $DC -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT" logs --tail=50 backend >&2
    exit 2
  fi
  ok "Backend ready, DB reset to clean seed state"
else
  warn "Skipping stack startup (--no-stack); assuming it's already up at $BASE_URL"
fi

# ---- 6. run tests ----
step "Running Playwright"
set +e
if (( UI_MODE == 1 )); then
  npx playwright test --ui
  TEST_EXIT=$?
else
  npx playwright test
  TEST_EXIT=$?
fi
set -e

if (( TEST_EXIT == 0 )); then
  ok "All tests passed"
else
  fail "Tests failed (exit $TEST_EXIT)"
fi

# ---- 7. report path ----
if [[ -d playwright-report ]]; then
  echo
  echo "Report: file://$SCRIPT_DIR/playwright-report/index.html"
  echo "  (open with: npx playwright show-report)"
fi

# ---- 8. tear down ----
if (( TEAR_DOWN == 1 )); then
  step "Tearing down stack"
  $DC -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT" down -v
  ok "Stack down, volumes removed"
fi

# Exit with the test exit code unless --keep-going.
if (( KEEP_GOING == 1 )); then
  exit 0
fi
exit $TEST_EXIT
