#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

ln -sf "${SCRIPT_DIR}/.tmux.conf" "${HOME}/.tmux.conf"

curl -fsSL https://claude.ai/install.sh | bash

# .envテンプレートのコピー
if [ ! -f "${SCRIPT_DIR}/.env" ]; then
  if [ -f "${SCRIPT_DIR}/.env-template" ]; then
    cp "${SCRIPT_DIR}/.env-template" "${SCRIPT_DIR}/.env"
    echo ".env file created from template."
  fi
fi

# ollamaにモデルをpull（未ダウンロードの場合のみ実行される）
echo "Pulling qwen3:1.7b model from Ollama..."
curl --fail --show-error --silent \
  --retry 10 --retry-delay 3 --retry-connrefused \
  http://ollama:11434/api/pull \
  -d '{"name": "qwen3:1.7b"}' > /dev/null
echo "Ollama model pull completed."
