#!/bin/bash

# セッション情報をstdinからJSON形式で受け取り、ステータスラインを表示する

INPUT=$(cat)

# 全ての値を一度のjq呼び出しで抽出
IFS=$'\t' read -r model context_size current_usage cost input_tokens output_tokens < <(
  echo "$INPUT" | jq -r '[
    .model // "",
    .contextWindow.totalSize // 0,
    .contextWindow.currentUsage // 0,
    .costUSD // "",
    ((.usage.inputTokens // 0) + (.usage.cacheCreationInputTokens // 0) + (.usage.cacheReadInputTokens // 0)),
    .usage.outputTokens // 0
  ] | @tsv'
)

format_tokens() {
  local tokens=$1
  if [ "$tokens" -ge 1000 ]; then
    local whole=$((tokens / 1000))
    local frac=$(( (tokens % 1000) / 100 ))
    echo "${whole}.${frac}k"
  else
    echo "$tokens"
  fi
}

# コンテキスト使用率の計算
if [ "$context_size" -eq 0 ] 2>/dev/null; then
  context_percent=0
else
  context_percent=$((current_usage * 100 / context_size))
fi

formatted_input=$(format_tokens "$input_tokens")
formatted_output=$(format_tokens "$output_tokens")

output="Model: ${model} | Context: ${context_percent}% | Tokens: ${formatted_input} in / ${formatted_output} out"

if [ -n "$cost" ]; then
  output="${output} | Cost: \$${cost}"
fi

echo "$output"
