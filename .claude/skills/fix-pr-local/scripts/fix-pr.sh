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

# リポジトリ名を git remote URL から取得（--repo フラグで使用する）
# github.com 直接アクセスとプロキシ経由（/git/owner/repo）の両方に対応
REMOTE_URL=$(git remote get-url origin 2>/dev/null || true)
if [[ "$REMOTE_URL" =~ github\.com[:/]([^/]+/[^/]+) ]]; then
  REPO="${BASH_REMATCH[1]}"
elif [[ "$REMOTE_URL" =~ /git/([^/]+/[^/]+) ]]; then
  REPO="${BASH_REMATCH[1]}"
else
  echo "ERROR: origin の remote URL から GitHub リポジトリを特定できません。" >&2
  echo "URL: ${REMOTE_URL:-<not set>}" >&2
  exit 1
fi
REPO="${REPO%.git}"

echo "=== Step 1: マージ状態の確認 ==="
PR_STATE=$(gh pr view "$PR_NUMBER" --repo "$REPO" --json state -q '.state')
if [ "$PR_STATE" = "MERGED" ]; then
  echo "PR #${PR_NUMBER} is already merged."
  exit 0
fi

echo "=== Step 2: ブランチの更新 ==="
gh pr update-branch "$PR_NUMBER" --repo "$REPO" 2>/dev/null || {
  echo "Branch update failed. Attempting merge from base branch..."
  BASE_BRANCH=$(gh pr view "$PR_NUMBER" --repo "$REPO" --json baseRefName -q '.baseRefName')
  gh pr checkout "$PR_NUMBER" --repo "$REPO"
  git fetch origin "$BASE_BRANCH"
  if ! git merge origin/"$BASE_BRANCH" --no-edit; then
    git merge --abort 2>/dev/null || true
    echo "Merge conflict detected."
    exit 1
  fi
  git push
}

echo "=== Step 3: マージ設定の確認 ==="
MERGE_METHODS=$(gh api "repos/${REPO}" --jq '{squash: .allow_squash_merge, merge: .allow_merge_commit, rebase: .allow_rebase_merge}')
echo "Available merge methods: ${MERGE_METHODS}"

echo "=== Step 4: auto-mergeの有効化 ==="
PREFERRED_METHOD=""
if echo "$MERGE_METHODS" | jq -e '.squash == true' > /dev/null 2>&1; then
  PREFERRED_METHOD="squash"
elif echo "$MERGE_METHODS" | jq -e '.merge == true' > /dev/null 2>&1; then
  PREFERRED_METHOD="merge"
elif echo "$MERGE_METHODS" | jq -e '.rebase == true' > /dev/null 2>&1; then
  PREFERRED_METHOD="rebase"
fi

if [ -n "$PREFERRED_METHOD" ]; then
  gh pr merge "$PR_NUMBER" --repo "$REPO" --auto --"$PREFERRED_METHOD" 2>/dev/null || true
else
  echo "No merge method is enabled for this repository." >&2
  exit 1
fi

echo "=== Step 5: CI完了の待機 ==="
MAX_POLLS=60
POLL_INTERVAL=15
for i in $(seq 1 $MAX_POLLS); do
  STATUS=$(gh pr checks "$PR_NUMBER" --repo "$REPO" --json state -q '
    [.[].state] as $s
    | if (($s | length) > 0) and ($s | all(. == "SUCCESS" or . == "SKIPPED")) then "success"
      elif ($s | any(. == "FAILURE")) then "failure"
      elif ($s | any(. == "PENDING" or . == "QUEUED")) then "pending"
      else "unknown"
      end
  ' 2>/dev/null || echo "pending")

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
PR_STATE=$(gh pr view "$PR_NUMBER" --repo "$REPO" --json state -q '.state')
if [ "$PR_STATE" = "MERGED" ]; then
  echo "PR #${PR_NUMBER} has been merged via auto-merge."
  exit 0
fi

echo "=== Step 7: ブロッカーの確認 ==="
# 未解決のレビュースレッドを確認（ページネーション対応）
UNRESOLVED=0
HAS_NEXT=true
CURSOR=""
while [ "$HAS_NEXT" = "true" ]; do
  AFTER_ARG=""
  if [ -n "$CURSOR" ]; then
    AFTER_ARG="-f cursor=$CURSOR"
  fi
  RESULT=$(gh api graphql -f query='
    query($owner: String!, $name: String!, $number: Int!, $cursor: String) {
      repository(owner: $owner, name: $name) {
        pullRequest(number: $number) {
          reviewThreads(first: 100, after: $cursor) {
            pageInfo { hasNextPage endCursor }
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
  ' -f owner="${REPO%%/*}" -f name="${REPO##*/}" -F number="$PR_NUMBER" $AFTER_ARG)

  PAGE_UNRESOLVED=$(echo "$RESULT" | jq '.data.repository.pullRequest.reviewThreads.nodes | map(select(.isResolved == false)) | length')
  UNRESOLVED=$((UNRESOLVED + PAGE_UNRESOLVED))
  HAS_NEXT=$(echo "$RESULT" | jq -r '.data.repository.pullRequest.reviewThreads.pageInfo.hasNextPage')
  CURSOR=$(echo "$RESULT" | jq -r '.data.repository.pullRequest.reviewThreads.pageInfo.endCursor // empty')
done

if [ "$UNRESOLVED" -gt 0 ]; then
  echo "${UNRESOLVED} unresolved review thread(s) found."
  exit 20
fi

# 承認の確認
# reviewDecisionが空の場合はレビュー不要の設定と判断し、マージを許可する
REVIEW_DECISION=$(gh pr view "$PR_NUMBER" --repo "$REPO" --json reviewDecision -q '.reviewDecision // empty')
if [ "$REVIEW_DECISION" != "APPROVED" ] && [ -n "$REVIEW_DECISION" ]; then
  echo "PR is not approved. Review decision: ${REVIEW_DECISION}"
  exit 30
fi

echo "=== Step 8: マージ実行 ==="
gh pr merge "$PR_NUMBER" --repo "$REPO" --"$PREFERRED_METHOD" --delete-branch || {
  echo "Merge failed."
  exit 1
}

echo "PR #${PR_NUMBER} has been merged successfully."
exit 0
