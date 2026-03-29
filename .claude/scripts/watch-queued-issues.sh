#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INTERVAL_SECONDS=60

REPO=$(gh repo view --json nameWithOwner -q '.nameWithOwner')

# シグナルハンドリング
shutdown() {
  echo ""
  echo "Shutting down..."
  exit 0
}
trap shutdown SIGINT

echo "Watching for queued issues in ${REPO}..."

while true; do
  # assign-to-claudeラベル付きかつin-progress-by-claude未付与のIssueを検索
  ISSUE=$(gh issue list --repo "$REPO" --label "assign-to-claude" --state open \
    --search "sort:created-asc" --json number,title,labels \
    -q '[.[] | select(.labels | map(.name) | contains(["in-progress-by-claude"]) | not)] | first // empty')

  if [ -n "$ISSUE" ]; then
    ISSUE_NUMBER=$(echo "$ISSUE" | jq -r '.number')
    ISSUE_TITLE=$(echo "$ISSUE" | jq -r '.title')
    echo ""
    echo "=========================================="
    echo "Processing Issue #${ISSUE_NUMBER}: ${ISSUE_TITLE}"
    echo "=========================================="
    "${SCRIPT_DIR}/solve-issue.sh" "$ISSUE_NUMBER" || true
    echo "Completed Issue #${ISSUE_NUMBER}."
  else
    printf "."
  fi

  sleep "$INTERVAL_SECONDS"
done
