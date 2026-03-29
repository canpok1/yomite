#!/usr/bin/env bash
set -euo pipefail

# handle-coderabbit-rate-limit.sh - CodeRabbitのレート制限を処理するスクリプト
#
# 使い方: handle-coderabbit-rate-limit.sh <PR_NUMBER> <WAIT_SECONDS>
#
# 終了コード:
#   0 : 新しいCodeRabbitレビューを検出
#   1 : タイムアウト（最大ポーリング回数に到達）

PR_NUMBER="${1:-}"
WAIT_SECONDS="${2:-0}"

POLL_INTERVAL="${POLL_INTERVAL:-30}"
MAX_POLLS="${MAX_POLLS:-30}"

if [ -z "$PR_NUMBER" ] || ! [[ "$PR_NUMBER" =~ ^[0-9]+$ ]]; then
  echo "Usage: $0 <PR_NUMBER> <WAIT_SECONDS>" >&2
  exit 1
fi

REPO=$(gh repo view --json nameWithOwner -q '.nameWithOwner')
CODERABBIT_USER="coderabbitai"

# 現在のCodeRabbitコメント・レビュー数を取得
BEFORE_COMMENTS=$(gh api "repos/${REPO}/pulls/${PR_NUMBER}/comments" --jq "[.[] | select(.user.login == \"${CODERABBIT_USER}\")] | length")
BEFORE_REVIEWS=$(gh api "repos/${REPO}/pulls/${PR_NUMBER}/reviews" --jq "[.[] | select(.user.login == \"${CODERABBIT_USER}\")] | length")

# レート制限の待機
if [ "$WAIT_SECONDS" -gt 0 ]; then
  echo "Waiting ${WAIT_SECONDS} seconds for CodeRabbit rate limit..."
  sleep "$WAIT_SECONDS"
fi

# レビューリクエストを投稿
echo "Requesting CodeRabbit review..."
gh pr comment "$PR_NUMBER" --repo "$REPO" --body "@coderabbitai review"

# レビュー開始前のバッファ待機
echo "Waiting 60 seconds for CodeRabbit to start processing..."
sleep 60

# ポーリング
for i in $(seq 1 $MAX_POLLS); do
  AFTER_COMMENTS=$(gh api "repos/${REPO}/pulls/${PR_NUMBER}/comments" --jq "[.[] | select(.user.login == \"${CODERABBIT_USER}\")] | length")
  AFTER_REVIEWS=$(gh api "repos/${REPO}/pulls/${PR_NUMBER}/reviews" --jq "[.[] | select(.user.login == \"${CODERABBIT_USER}\")] | length")

  if [ "$AFTER_COMMENTS" -gt "$BEFORE_COMMENTS" ] || [ "$AFTER_REVIEWS" -gt "$BEFORE_REVIEWS" ]; then
    echo "New CodeRabbit activity detected."
    exit 0
  fi

  echo "Polling for CodeRabbit response... (${i}/${MAX_POLLS})"
  sleep "$POLL_INTERVAL"
done

echo "Timeout: No new CodeRabbit activity after ${MAX_POLLS} polls."
exit 1
