#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_DIR="${WORKSPACE_DIR:-$(cd "${SCRIPT_DIR}/../.." && pwd)}"

# オプション解析
PRINT_MODE=false
while getopts "p" opt; do
  case $opt in
    p) PRINT_MODE=true ;;
    *) echo "Usage: $0 [-p] <issue_number>" >&2; exit 1 ;;
  esac
done
shift $((OPTIND - 1))

ISSUE_NUMBER="${1:-}"
if [ -z "$ISSUE_NUMBER" ] || ! [[ "$ISSUE_NUMBER" =~ ^[0-9]+$ ]]; then
  echo "Error: issue_number must be numeric" >&2
  exit 1
fi

REPO=$(gh repo view --json nameWithOwner -q '.nameWithOwner')

# ロックファイル
LOCK_DIR="${WORKSPACE_DIR}/.tmp/locks"
mkdir -p "$LOCK_DIR"
LOCK_FILE="${LOCK_DIR}/issue-${ISSUE_NUMBER}.lock"

if [ -f "$LOCK_FILE" ]; then
  echo "Issue #${ISSUE_NUMBER} is already being processed by another process." >&2
  exit 1
fi

touch "$LOCK_FILE"

# クリーンアップ（失敗時のみラベル除去、成功時はsolve-issueスキルがPRでクローズ）
SOLVE_SUCCESS=false
cleanup() {
  rm -f "$LOCK_FILE"
  if [ "$SOLVE_SUCCESS" = false ]; then
    gh issue edit "$ISSUE_NUMBER" --repo "$REPO" --remove-label "in-progress-by-claude" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# ラベル付与
gh issue edit "$ISSUE_NUMBER" --repo "$REPO" --add-label "in-progress-by-claude"

# メインブランチに切り替え
cd "$WORKSPACE_DIR"
git stash --include-untracked 2>/dev/null || true
git checkout main
git pull origin main

# Claude実行
PROMPT="/solve-issue ${ISSUE_NUMBER}"
if [ "$PRINT_MODE" = true ]; then
  "${SCRIPT_DIR}/claude-stream.sh" --worktree -p "$PROMPT"
else
  claude --worktree -p "$PROMPT"
fi

SOLVE_SUCCESS=true
echo "Issue #${ISSUE_NUMBER} processing completed."
