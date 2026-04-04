FROM mcr.microsoft.com/devcontainers/base:jammy

ENV CLAUDE_CONFIG_DIR=/home/vscode/.claude

RUN apt-get update && apt-get install -y --no-install-recommends \
    jq \
    tmux \
    shellcheck \
    libgtk-3-dev \
    libwebkit2gtk-4.0-dev \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /workspaces && chown vscode:vscode /workspaces

USER vscode

RUN mkdir -p ${CLAUDE_CONFIG_DIR} && chmod 700 ${CLAUDE_CONFIG_DIR}
