#!/usr/bin/env bash
# stop-check.sh
# Stop hook：
#   1. 若有未提交變更        → exit 2，喚醒 Claude 要求 commit + changelog
#   2. 若 session 內有新 commit
#      但沒有碰過 changelogs/ → exit 2，喚醒 Claude 補寫更動報告

INPUT=$(cat)
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"' 2>/dev/null || echo "unknown")

git rev-parse --git-dir > /dev/null 2>&1 || exit 0

# ── 1. 未提交變更 ──────────────────────────────────────────────────────────
if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
  echo "仍有未提交的變更，請依照 staged-commit 工作流程進行分層 commit，並撰寫更動報告（changelogs/YYYY-MM-DD_描述.md）"
  exit 2
fi

# ── 2. 有新 commit 但沒有 changelog ───────────────────────────────────────
HEAD_FILE="/tmp/claude-session-${SESSION_ID}.head"

if [ -f "$HEAD_FILE" ]; then
  START=$(cat "$HEAD_FILE")
  NOW=$(git rev-parse HEAD 2>/dev/null)

  if [ "$START" != "$NOW" ]; then
    CHANGELOG_COUNT=$(git log "${START}..HEAD" --name-only --format="" -- changelogs/ 2>/dev/null | grep -c "." 2>/dev/null || echo 0)
    if [ "$CHANGELOG_COUNT" -eq 0 ]; then
      echo "本次 session 有新 commit 但缺少更動報告，請補寫 changelogs/YYYY-MM-DD_描述.md 並 commit"
      exit 2
    fi
  fi
fi

exit 0
