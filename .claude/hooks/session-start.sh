#!/usr/bin/env bash
# session-start.sh
# SessionStart hook：把目前的 git HEAD 存到 /tmp，
# 讓 stop-check.sh 在 session 結束時比對有沒有新 commit。

INPUT=$(cat)
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"' 2>/dev/null || echo "unknown")

git rev-parse --git-dir > /dev/null 2>&1 || exit 0
git rev-parse HEAD > "/tmp/claude-session-${SESSION_ID}.head" 2>/dev/null || true

exit 0
