FROM python:3.12-slim

# Install system dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      curl ca-certificates git jq unzip sudo && \
    curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Install Claude Code CLI and Google Workspace CLI
RUN npm install -g @anthropic-ai/claude-code @googleworkspace/cli

# Install Python dependencies
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Create non-root user with passwordless sudo access.
# Non-root is required by Claude CLI for --dangerously-skip-permissions.
# Sudo allows agents to install packages at runtime (apt-get, pip, npm).
RUN useradd -m -s /bin/bash komputer && \
    echo "komputer ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers && \
    mkdir -p /workspace && chown komputer:komputer /workspace
USER komputer

WORKDIR /workspace

# Route package installs to the persistent workspace so they survive pod restarts.
# pip: installs to /workspace/.local (via --user default)
# npm: global installs to /workspace/.npm-global
# PATH: includes both so installed binaries are found
ENV PYTHONUNBUFFERED=1 \
    PIP_USER=1 \
    PYTHONUSERBASE=/workspace/.local \
    NPM_CONFIG_PREFIX=/workspace/.npm-global \
    PATH="/workspace/.local/bin:/workspace/.npm-global/bin:${PATH}"

ENTRYPOINT ["python", "/app/main.py"]
