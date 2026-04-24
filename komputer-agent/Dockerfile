FROM python:3.12-slim

# Install system dependencies (no Node.js — Claude Code is bundled in the SDK)
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      curl ca-certificates git jq unzip sudo && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Install Google Workspace CLI binary from GitHub releases (no npm/node needed)
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "amd64" ]; then TRIPLE="x86_64-unknown-linux-gnu"; \
    elif [ "$ARCH" = "arm64" ]; then TRIPLE="aarch64-unknown-linux-gnu"; \
    else echo "Unsupported arch: $ARCH" && exit 1; fi && \
    curl -fsSL "https://github.com/googleworkspace/cli/releases/download/v0.22.5/google-workspace-cli-${TRIPLE}.tar.gz" | \
    tar -xzf - -C /usr/local/bin ./gws && \
    chmod +x /usr/local/bin/gws

# Install Python dependencies (libsnappy-dev required for python-snappy)
WORKDIR /app
COPY requirements.txt .
RUN apt-get update && apt-get install -y --no-install-recommends libsnappy-dev && \
    rm -rf /var/lib/apt/lists/* && \
    pip install --no-cache-dir -r requirements.txt && \
    pip cache purge 2>/dev/null; rm -rf /root/.cache/pip

COPY . .

# Create non-root user with passwordless sudo access.
# Non-root is required by Claude CLI for --dangerously-skip-permissions.
# Sudo allows agents to install packages at runtime (apt-get, pip, npm).
RUN useradd -m -s /bin/bash komputer && \
    echo "komputer ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers && \
    mkdir -p /workspace /files && chown komputer:komputer /workspace /files
USER komputer

WORKDIR /workspace

# Route package installs to the persistent workspace so they survive pod restarts.
# pip: installs to /workspace/.local (via --user default)
# npm: global installs to /workspace/.npm-global (if agent installs node later)
# PATH: includes both so installed binaries are found
ENV PYTHONUNBUFFERED=1 \
    PIP_USER=1 \
    PYTHONUSERBASE=/workspace/.local \
    NPM_CONFIG_PREFIX=/workspace/.npm-global \
    PATH="/workspace/.local/bin:/workspace/.npm-global/bin:${PATH}"

ENTRYPOINT ["python", "/app/main.py"]
