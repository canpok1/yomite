#!/usr/bin/env bash
set -euo pipefail

# fix-pr.sh - PRのマージプロセスを自動化するスクリプト
#
# 使い方: fix-pr.sh <PR_NUMBER>
#
# 終了コード:
#   0  : マージ完了
#   1  : スクリプトエラー（コンフリクト含む）
#   10 : CI失敗
#   20 : 未解決のレビュースレッドあり
#   30 : CI通過・スレッド解決済みだが承認なし

PR_NUMBER="${1:-}"
if [ -z "$PR_NUMBER" ] || ! [[ "$PR_NUMBER" =~ ^[0-9]+$ ]]; then
  echo "Usage: $0 <PR_NUMBER>" >&2
  exit 1
fi

REPO=$(gh repo view --json nameWithOwner -q '.nameWithOwner')

echo "=== Step 1: マージ状態の確認 ==="
MERGED=$(gh pr view "$PR_NUMBER" --repo "$REPO" --json merged -q '.merged')
if [ "$MERGED" = "true" ]; then
  echo "PR #${PR_NUMBER} is already merged."
  exit 0
fi

echo "=== Step 2: ブランチの更新 ==="
gh pr update-branch "$PR_NUMBER" --repo "$REPO" 2>/dev/null || {
  echo "Branch update failed. Attempting merge from main..."
  PR_BRANCH=$(gh pr view "$PR_NUMBER" --repo "$REPO" --json headRefName -q '.headRefName')
  git fetch origin main "$PR_BRANCH"
  git checkout "$PR_BRANCH"
  if ! git merge origin/main --no-edit; then
    git merge --abort 2>/dev/null || true
    echo "Merge conflict detected."
    exit 1
  fi
  git push origin "$PR_BRANCH"
}

echo "=== Step 3: マージ設定の確認 ==="
MERGE_METHODS=$(gh api "repos/${REPO}" --jq '{squash: .allow_squash_merge, merge: .allow_merge_commit, rebase: .allow_rebase_merge}')
echo "Available merge methods: ${MERGE_METHODS}"

echo "=== Step 4: auto-mergeの有効化 ==="
PREFERRED_METHOD="SQUASH"
if echo "$MERGE_METHODS" | jq -e '.squash' > /dev/null 2>&1; then
  PREFERRED_METHOD="SQUASH"
elif echo "$MERGE_METHODS" | jq -e '.merge' > /dev/null 2>&1; then
  PREFERRED_METHOD="MERGE"
elif echo "$MERGE_METHODS" | jq -e '.rebase' > /dev/null 2>&1; then
  PREFERRED_METHOD="REBASE"
fi
gh pr merge "$PR_NUMBER" --repo "$REPO" --auto --"$(echo "$PREFERRED_METHOD" | tr '[:upper:]' '[:lower:]')" 2>/dev/null || true

echo "=== Step 5: CI完了の待機 ==="
MAX_POLLS=60
POLL_INTERVAL=15
for i in $(seq 1 $MAX_POLLS); do
  STATUS=$(gh pr checks "$PR_NUMBER" --repo "$REPO" --json state -q '[.[].state] | if all(. == "SUCCESS" or . == "SKIPPED") then "success" elif any(. == "FAILURE") then "failure" elif any(. == "PENDING" or . == "QUEUED") then "pending" else "unknown" end' 2>/dev/null || echo "pending")

  case "$STATUS" in
    success)
      echo "All CI checks passed."
      break
      ;;
    failure)
      echo "CI checks failed."
      exit 10
      ;;
    pending|unknown)
      echo "Waiting for CI... (${i}/${MAX_POLLS})"
      sleep "$POLL_INTERVAL"
      ;;
  esac
done

if [ "$STATUS" = "pending" ] || [ "$STATUS" = "unknown" ]; then
  echo "CI timed out."
  exit 10
fi

echo "=== Step 6: マージ状態の再確認 ==="
MERGED=$(gh pr view "$PR_NUMBER" --repo "$REPO" --json merged -q '.merged')
if [ "$MERGED" = "true" ]; then
  echo "PR #${PR_NUMBER} has been merged via auto-merge."
  exit 0
fi

echo "=== Step 7: ブロッカーの確認 ==="
# 未解決のレビュースレッドを確認
UNRESOLVED=$(gh api graphql -f query='
  query($owner: String!, $name: String!, $number: Int!) {
    repository(owner: $owner, name: $name) {
      pullRequest(number: $number) {
        reviewThreads(first: 100) {
          nodes {
            isResolved
            comments(first: 1) {
              nodes {
                author { login }
                body
              }
            }
          }
        }
      }
    }
  }
' -f owner="${REPO%%/*}" -f name="${REPO##*/}" -F number="$PR_NUMBER" \
  --jq '.data.repository.pullRequest.reviewThreads.nodes | map(select(.isResolved == false)) | length')

if [ "$UNRESOLVED" -gt 0 ]; then
  echo "${UNRESOLVED} unresolved review thread(s) found."
  exit 20
fi

# 承認の確認
REVIEW_DECISION=$(gh pr view "$PR_NUMBER" --repo "$REPO" --json reviewDecision -q '.reviewDecision')
if [ "$REVIEW_DECISION" != "APPROVED" ] && [ -n "$REVIEW_DECISION" ]; then
  echo "PR is not approved. Review decision: ${REVIEW_DECISION}"
  exit 30
fi

echo "=== Step 8: マージ実行 ==="
gh pr merge "$PR_NUMBER" --repo "$REPO" --"$(echo "$PREFERRED_METHOD" | tr '[:upper:]' '[:lower:]')" --delete-branch || {
  echo "Merge failed."
  exit 1
}

echo "PR #${PR_NUMBER} has been merged successfully."
exit 0
