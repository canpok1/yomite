#!/bin/bash

# セッション情報をstdinからJSON形式で受け取り、ステータスラインを表示する

INPUT=$(cat)

get_model_name() {
  echo "$INPUT" | jq -r '.model // empty'
}

get_context_window_size() {
  echo "$INPUT" | jq -r '.contextWindow.totalSize // 0'
}

get_current_usage() {
  echo "$INPUT" | jq -r '.contextWindow.currentUsage // 0'
}

get_cost() {
  echo "$INPUT" | jq -r '.costUSD // empty'
}

get_input_tokens() {
  local normal cache_creation cache_read
  normal=$(echo "$INPUT" | jq -r '.usage.inputTokens // 0')
  cache_creation=$(echo "$INPUT" | jq -r '.usage.cacheCreationInputTokens // 0')
  cache_read=$(echo "$INPUT" | jq -r '.usage.cacheReadInputTokens // 0')
  echo $((normal + cache_creation + cache_read))
}

get_output_tokens() {
  echo "$INPUT" | jq -r '.usage.outputTokens // 0'
}

format_tokens() {
  local tokens=$1
  if [ "$tokens" -ge 1000 ]; then
    printf "%.1fk" "$(echo "scale=1; $tokens / 1000" | bc)"
  else
    echo "$tokens"
  fi
}

calc_context_percent() {
  local usage=$1 total=$2
  if [ "$total" -eq 0 ]; then
    echo "0"
  else
    echo $((usage * 100 / total))
  fi
}

# メイン処理
model=$(get_model_name)
context_size=$(get_context_window_size)
current_usage=$(get_current_usage)
context_percent=$(calc_context_percent "$current_usage" "$context_size")
input_tokens=$(format_tokens "$(get_input_tokens)")
output_tokens=$(format_tokens "$(get_output_tokens)")
cost=$(get_cost)

output="Model: ${model} | Context: ${context_percent}% | Tokens: ${input_tokens} in / ${output_tokens} out"

if [ -n "$cost" ]; then
  output="${output} | Cost: \$${cost}"
fi

echo "$output"
