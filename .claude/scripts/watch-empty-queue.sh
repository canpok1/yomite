#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INTERVAL_SECONDS=60

# オプション解析
ASSIGN_COUNT=1
MIN_QUEUE=1
while [[ $# -gt 0 ]]; do
  case $1 in
    --assign-count) ASSIGN_COUNT="$2"; shift 2 ;;
    --min-queue) MIN_QUEUE="$2"; shift 2 ;;
    *) echo "Usage: $0 [--assign-count N] [--min-queue N]" >&2; exit 1 ;;
  esac
done

# ロックファイルによる多重起動防止（flockでアトミックなロック取得）
LOCK_FILE="/tmp/watch-empty-queue.lock"
exec 200>"$LOCK_FILE"
if ! flock -n 200; then
  echo "Already running."
  exit 1
fi

# シグナルハンドリング
shutdown() {
  echo ""
  echo "Shutting down..."
  exit 0
}
trap shutdown SIGINT

REPO=$(gh repo view --json nameWithOwner -q '.nameWithOwner')

echo "Watching for empty queue in ${REPO} (min_queue: ${MIN_QUEUE}, assign_count: ${ASSIGN_COUNT})..."

waiting=false
while true; do
  # assign-to-claudeラベル付きのオープンIssue数を確認
  QUEUE_COUNT=$(gh issue list --repo "$REPO" --label "assign-to-claude" --state open --json number -q 'length')

  if [ "$QUEUE_COUNT" -lt "$MIN_QUEUE" ]; then
    if [ "$waiting" = true ]; then
      echo ""
    fi
    waiting=false
    echo "Queue count (${QUEUE_COUNT}) is below minimum (${MIN_QUEUE}). Assigning issues..."
    "${SCRIPT_DIR}/assign-issues.sh" -c "$ASSIGN_COUNT" || true
  else
    if [ "$waiting" = false ]; then
      printf "Waiting for queue to empty"
    fi
    printf "."
    waiting=true
  fi

  sleep "$INTERVAL_SECONDS"
done
