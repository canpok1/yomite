#!/bin/bash

set -euo pipefail

TEMP_DIR=""

cleanup() {
  if [[ -n "${TEMP_DIR}" ]] && [[ -d "${TEMP_DIR}" ]]; then
    rm -rf "${TEMP_DIR}"
  fi
}

trap cleanup EXIT

if command -v gh &> /dev/null; then
  echo "gh is already installed: $(gh --version)"
  exit 0
fi

VERSION="${GH_SETUP_VERSION:-2.83.2}"
if [[ ! "${VERSION}" =~ ^[0-9.]+$ ]]; then
  echo "Error: Invalid version format: ${VERSION}" >&2
  exit 1
fi

# GitHub CLIのリリースはx86_64をamd64、aarch64をarm64として配布しているため変換
ARCH=$(uname -m)
case "${ARCH}" in
  x86_64)
    ARCH="amd64"
    ;;
  aarch64|arm64)
    ARCH="arm64"
    ;;
  *)
    echo "Error: Unsupported architecture: ${ARCH}" >&2
    exit 1
    ;;
esac

# macOSはリリースアセット名が異なる（darwin→macOS、.tar.gz→.zip）
RAW_OS=$(uname -s)
case "${RAW_OS}" in
  Linux)
    OS="linux"
    ARCHIVE_EXT="tar.gz"
    ;;
  Darwin)
    OS="macOS"
    ARCHIVE_EXT="zip"
    ;;
  *)
    echo "Error: Unsupported OS: ${RAW_OS}" >&2
    exit 1
    ;;
esac

ARCHIVE_NAME="gh_${VERSION}_${OS}_${ARCH}.${ARCHIVE_EXT}"
URL="https://github.com/cli/cli/releases/download/v${VERSION}/${ARCHIVE_NAME}"

echo "Downloading gh ${VERSION} for ${OS}/${ARCH}..."

TEMP_DIR=$(mktemp -d)

cd "${TEMP_DIR}"
if ! curl -fsSL "${URL}" -o "${ARCHIVE_NAME}"; then
  echo "Error: Failed to download from ${URL}" >&2
  exit 1
fi

if [[ "${ARCHIVE_EXT}" == "zip" ]]; then
  if ! unzip -q "${ARCHIVE_NAME}"; then
    echo "Error: Failed to extract ${ARCHIVE_NAME}" >&2
    exit 1
  fi
else
  if ! tar -xzf "${ARCHIVE_NAME}"; then
    echo "Error: Failed to extract ${ARCHIVE_NAME}" >&2
    exit 1
  fi
fi

GH_BINARY="gh_${VERSION}_${OS}_${ARCH}/bin/gh"
if [[ ! -f "${GH_BINARY}" ]]; then
  echo "Error: gh binary not found in downloaded archive at expected path: ./${GH_BINARY}" >&2
  exit 1
fi

INSTALL_DIR="${HOME}/.local/bin"
mkdir -p "${INSTALL_DIR}"

cp "${GH_BINARY}" "${INSTALL_DIR}/gh"
chmod 755 "${INSTALL_DIR}/gh"

echo "gh installed to ${INSTALL_DIR}/gh"

export PATH="${INSTALL_DIR}:${PATH}"

# セッション間でPATHを維持するため環境ファイルに永続化
if [[ -n "${CLAUDE_ENV_FILE:-}" ]]; then
  if ! grep -q "export PATH=.*${INSTALL_DIR}" "${CLAUDE_ENV_FILE}" 2>/dev/null; then
    echo "export PATH=\"${INSTALL_DIR}:\${PATH}\"" >> "${CLAUDE_ENV_FILE}"
    echo "PATH setting persisted to ${CLAUDE_ENV_FILE}"
  fi
else
  echo "Warning: CLAUDE_ENV_FILE is not set. PATH will not be persisted across sessions." >&2
fi

if [[ -n "${GH_TOKEN:-}" ]] || [[ -n "${GITHUB_TOKEN:-}" ]]; then
  echo "Setting up Git authentication..."
  gh auth setup-git
fi

echo "gh installation completed successfully!"
gh --version
