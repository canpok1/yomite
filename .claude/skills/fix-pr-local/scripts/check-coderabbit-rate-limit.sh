#!/usr/bin/env bash
set -euo pipefail

# check-coderabbit-rate-limit.sh - CodeRabbitのレート制限コメントが有効かチェックするスクリプト
#
# 使い方: check-coderabbit-rate-limit.sh <PR_NUMBER> <REPO>
#
# 出力:
#   レート制限が有効な場合: 残り待機秒数を stdout に出力し、exit 0
#   レート制限なし/期限切れの場合: 何も出力せず、exit 1
#
# 判定ロジック:
#   1. coderabbitai の全 issue コメントから updatedAt が最新のものを取得
#   2. そのコメントがレート制限メッセージか判定
#   3. レート制限コメントの場合、updatedAt + 待機秒数 < 現在時刻 なら期限切れ

PR_NUMBER="${1:-}"
REPO="${2:-}"

if [ -z "$PR_NUMBER" ] || [ -z "$REPO" ]; then
  echo "Usage: $0 <PR_NUMBER> <REPO>" >&2
  exit 1
fi

OWNER="${REPO%%/*}"
NAME="${REPO##*/}"

# GraphQL で coderabbitai の issue コメント（PR のトップレベルコメント）を取得
# updatedAt でソートするため全件取得して jq で最新を選ぶ
# NOTE: reviewThreads のコメントではなく issue comments を対象にする。
# CodeRabbit はレート制限時にサマリーの issue comment を書き換えるため。
RESULT=$(gh api graphql -f query='
  query($owner: String!, $name: String!, $number: Int!) {
    repository(owner: $owner, name: $name) {
      pullRequest(number: $number) {
        comments(first: 100) {
          nodes {
            author { login }
            body
            updatedAt
          }
        }
      }
    }
  }
' -f owner="$OWNER" -f name="$NAME" -F number="$PR_NUMBER")

# coderabbitai のコメントのうち updatedAt が最新のものを取得
LATEST_COMMENT=$(echo "$RESULT" | jq -r '
  [.data.repository.pullRequest.comments.nodes[]
   | select(.author.login == "coderabbitai[bot]")]
  | sort_by(.updatedAt)
  | last
  // empty
')

if [ -z "$LATEST_COMMENT" ] || [ "$LATEST_COMMENT" = "null" ]; then
  exit 1
fi

BODY=$(echo "$LATEST_COMMENT" | jq -r '.body')
UPDATED_AT=$(echo "$LATEST_COMMENT" | jq -r '.updatedAt')

# レート制限メッセージの判定
# CodeRabbit のレート制限メッセージには "rate limit" や "exhausted" が含まれる
if ! echo "$BODY" | grep -qiE '(rate limit|API rate limit exhausted|exceeded.*rate)'; then
  exit 1
fi

# 待機秒数をコメント本文からパース
# 例: "Please retry after 1234 seconds" や "retry in 1234 seconds"
WAIT_SECONDS=$(echo "$BODY" | grep -oP '(?:retry (?:after|in) |wait )\K[0-9]+(?= seconds?)' | head -1)
if [ -z "$WAIT_SECONDS" ]; then
  # 秒数が取得できない場合はデフォルト 3600 秒（1時間）
  WAIT_SECONDS=3600
fi

# 有効期限チェック: updatedAt + WAIT_SECONDS < 現在時刻 なら期限切れ
UPDATED_EPOCH=$(date -d "$UPDATED_AT" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%SZ" "$UPDATED_AT" +%s 2>/dev/null)
NOW_EPOCH=$(date +%s)
EXPIRY_EPOCH=$((UPDATED_EPOCH + WAIT_SECONDS))

if [ "$NOW_EPOCH" -ge "$EXPIRY_EPOCH" ]; then
  echo "Rate limit comment found but expired (updated: ${UPDATED_AT}, wait: ${WAIT_SECONDS}s)." >&2
  exit 1
fi

REMAINING=$((EXPIRY_EPOCH - NOW_EPOCH))
echo "$REMAINING"
exit 0
