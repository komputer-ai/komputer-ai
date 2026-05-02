---
title: Custom Agent Images
---

The default agent image (`komputer-agent`) ships with Python 3.12, git, curl, jq, the Claude Agent SDK (which bundles the Claude Code CLI as a native binary), and the Google Workspace CLI. Agents have sudo access and can install anything at runtime — `apt-get`, `pip`, `npm`, `cargo`, `go install`, downloading binaries, compiling from source, or any other method available on a Debian-based system. Package installs via `pip` and `npm` persist across tasks thanks to the workspace PVC.

Only `pip` and `npm` installs are persisted to the workspace PVC. System-level installs (`apt-get`, `cargo install`, downloaded binaries outside `/workspace`, etc.) live on the container filesystem and are **lost when the pod restarts**. If your agents consistently need specific tools or system packages across tasks, you should bake them into a custom agent image rather than relying on runtime installs.

## Extending the base image (recommended)

Build your own image using the official komputer-agent as the base:

```dockerfile
FROM ghcr.io/komputer-ai/komputer-agent:latest

# Switch to root to install system packages
USER root

# Install system-level tools
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      postgresql-client \
      awscli \
      ffmpeg && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Install Python packages globally (available to all agents)
RUN pip install --no-cache-dir pandas numpy boto3

# Switch back to non-root user (required for Claude CLI)
USER komputer
```

Build and use it:

```bash
docker build -t my-custom-agent:latest .
```

Then reference it in your template:

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgentClusterTemplate
metadata:
  name: custom
spec:
  anthropicKeySecretRef:
    name: anthropic-api-key
    key: api-key
    # namespace: komputer-ai  # optional — defaults to the install namespace
  podSpec:
    containers:
      - name: agent
        image: my-custom-agent:latest
  storage:
    size: "10Gi"
```

The `anthropicKeySecretRef` is required. The operator mirrors that Secret into every agent namespace and injects `ANTHROPIC_API_KEY` into the pod automatically — don't add it to `podSpec.env` yourself.

Agents can reference this template with `templateRef: "custom"`.

## What the base image includes

| Layer | Contents |
|-------|----------|
| OS | Debian (python:3.12-slim) |
| System tools | git, curl, jq, unzip, sudo |
| Claude Code CLI | Bundled as a native binary inside the Claude Agent SDK (no Node.js required) |
| Google Workspace CLI | Standalone Rust binary (`gws`) for Google services |
| Python | 3.12 + agent runtime (FastAPI, Claude Agent SDK, Redis client) |
| User | `komputer` (non-root, passwordless sudo) |
| Entrypoint | `python /app/main.py` |
| Image size | ~550MB |

Runtime package installs by agents persist to the workspace PVC:
- `pip install` goes to `/workspace/.local`
- `npm install -g` goes to `/workspace/.npm-global` (if agents install Node.js at runtime)
- Both are on `PATH` automatically

> **Note:** Node.js is not pre-installed in the base image. Agents can install it at runtime via `sudo apt-get install -y nodejs` if needed for a task. The Claude Code CLI does not require Node.js — it's bundled as a native binary.

## Using a completely different base image

If you need a different Linux distribution (e.g., Ubuntu, Alpine, RHEL), you can use the [agent Dockerfile](../komputer-agent/Dockerfile) as a reference and rebuild from scratch. Do that at your own risk — you're responsible for ensuring all dependencies are present and the agent runtime works correctly.

## Important constraints

When building a custom image (whether extending the base or rebuilding from scratch), keep these requirements in mind:

- **Non-root user** — The Claude CLI requires a non-root user with `--dangerously-skip-permissions`. The base image creates a `komputer` user for this. If you change the user, make sure it's non-root with sudo access.
- **Claude Agent SDK** — Must be installed via pip (`claude-agent-sdk`). The SDK bundles the Claude Code CLI as a native binary — no separate installation needed.
- **Entrypoint** — Must be `python /app/main.py`. The agent runtime (FastAPI server, event publisher, Claude SDK integration) lives in `/app/`. Do not override the entrypoint unless you know what you're doing.
- **Workspace at `/workspace`** — The operator mounts the persistent volume here. Agents work in this directory.
- **Python packages** — `requirements.txt` must be installed (FastAPI, Claude Agent SDK, Redis client, etc.)
