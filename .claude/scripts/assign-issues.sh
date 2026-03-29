#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# オプション解析
PRINT_MODE=false
ASSIGN_COUNT=2
while getopts "pc:" opt; do
  case $opt in
    p) PRINT_MODE=true ;;
    c) ASSIGN_COUNT="$OPTARG" ;;
    *) echo "Usage: $0 [-p] [-c count]" >&2; exit 1 ;;
  esac
done

if ! [[ "$ASSIGN_COUNT" =~ ^[0-9]+$ ]]; then
  echo "Error: assign count must be numeric" >&2
  exit 1
fi

ME=$(gh api user -q '.login')
REPO=$(gh repo view --json nameWithOwner -q '.nameWithOwner')

# readyラベル付きかつ割り当て済みラベルなしのIssue数を確認し、0件ならスキップ
READY_COUNT=$(gh issue list --repo "$REPO" --label "ready" --state open --json number \
  -q '[.[] | select(.labels | map(.name) | (contains(["assign-to-claude"]) | not) and (contains(["in-progress-by-claude"]) | not))] | length')

if [ "$READY_COUNT" -eq 0 ]; then
  echo "No assignable issues found. Skipping."
  exit 0
fi

# Claude実行
PROMPT="/assign-issues --count ${ASSIGN_COUNT}"
if [ "$PRINT_MODE" = true ]; then
  "${SCRIPT_DIR}/claude-stream.sh" -p "$PROMPT"
else
  claude -p "$PROMPT"
fi
