#!/bin/bash

# セッション情報をstdinからJSON形式で受け取り、ステータスラインを表示する

INPUT=$(cat)

# 全ての値を一度のjq呼び出しで抽出
IFS=$'\t' read -r model context_percent cost input_tokens output_tokens < <(
  echo "$INPUT" | jq -r '[
    .model.display_name // "",
    .context_window.used_percentage // 0,
    .cost.total_cost_usd // "",
    .context_window.total_input_tokens // 0,
    .context_window.total_output_tokens // 0
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

# コンテキスト使用率（小数点以下を切り捨て）
context_percent=${context_percent%.*}

formatted_input=$(format_tokens "$input_tokens")
formatted_output=$(format_tokens "$output_tokens")

output="Model: ${model} | Context: ${context_percent}% | Tokens: ${formatted_input} in / ${formatted_output} out"

if [ -n "$cost" ]; then
  cost=$(printf '%.2f' "$cost")
  output="${output} | Cost: \$${cost}"
fi

echo "$output"
