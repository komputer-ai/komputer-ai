# komputer-agent

Python runtime for Claude AI agents. Runs inside Kubernetes pods created by the komputer-operator. Each agent has a persistent workspace, bash and web search tools, and publishes all activity as structured events to Redis.

## How It Works

1. **Startup** — Reads configuration from env vars and `/etc/komputer/config.json`
2. **Initial task** — Runs the task from `KOMPUTER_INSTRUCTIONS` in a background thread
3. **FastAPI server** — Stays running on port 8000, accepting new tasks via `POST /task`
4. **Events** — All agent activity (tool calls, messages, completions) is published to Redis
5. **Cancellation** — Running tasks can be cancelled via `POST /cancel`

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/task` | Submit a new task (blocked if busy) |
| `POST` | `/cancel` | Cancel the running task |
| `GET` | `/status` | Check if agent is busy |

### POST /task

```json
{
  "instructions": "Search the web for AI news",
  "model": "claude-sonnet-4-20250514"
}
```

`model` is optional — defaults to the value from `KOMPUTER_MODEL` env var.

Returns `409` if the agent is already executing a task.

### POST /cancel

Gracefully cancels the running asyncio task. The Claude CLI subprocess is terminated and a `task_cancelled` event is published.

## Event Format

Events are published to a Redis list as JSON:

```json
{"agentName": "my-agent", "type": "task_started", "timestamp": "2026-03-26T10:00:00Z", "payload": {"instructions": "..."}}
{"agentName": "my-agent", "type": "thinking", "timestamp": "...", "payload": {"content": "Reasoning about..."}}
{"agentName": "my-agent", "type": "tool_call", "timestamp": "...", "payload": {"id": "...", "tool": "WebSearch", "input": {"query": "..."}}}
{"agentName": "my-agent", "type": "tool_result", "timestamp": "...", "payload": {"tool": "WebSearch", "output": "..."}}
{"agentName": "my-agent", "type": "text", "timestamp": "...", "payload": {"content": "The answer is..."}}
{"agentName": "my-agent", "type": "task_completed", "timestamp": "...", "payload": {"result": "...", "cost_usd": 0.08, "duration_ms": 30000, "turns": 2, "stop_reason": "end_turn"}}
{"agentName": "my-agent", "type": "task_cancelled", "timestamp": "...", "payload": {"reason": "Cancelled by user"}}
{"agentName": "my-agent", "type": "error", "timestamp": "...", "payload": {"error": "..."}}
```

## Configuration

### Environment Variables (injected by the operator)

| Variable | Description |
|----------|-------------|
| `KOMPUTER_INSTRUCTIONS` | Initial task prompt |
| `KOMPUTER_MODEL` | Claude model (e.g. `claude-sonnet-4-20250514`) |
| `KOMPUTER_AGENT_NAME` | Agent identifier for events |
| `ANTHROPIC_API_KEY` | Anthropic API key (from template env/secret) |

### Config File

Mounted by the operator at `/etc/komputer/config.json`:

```json
{
  "redis": {
    "address": "redis:6379",
    "password": "",
    "db": 0,
    "queue": "komputer-events"
  }
}
```

## Claude Agent SDK

Uses the [Claude Agent SDK](https://pypi.org/project/claude-agent-sdk/) which wraps the Claude Code CLI. The agent is configured with:

- **Tools:** `Bash`, `WebSearch`
- **Permission mode:** `bypassPermissions` (requires non-root user)
- **Working directory:** `/workspace` (persistent via PVC)

The SDK requires the `claude` CLI binary, which is installed via `npm install -g @anthropic-ai/claude-code` in the Dockerfile.

## Development

### Prerequisites

- Python 3.12+
- Node.js 22+ (for Claude Code CLI)

### Local setup

```bash
pip install -r requirements.txt
npm install -g @anthropic-ai/claude-code

# Create a config file
echo '{"redis":{"address":"localhost:6379","db":0,"queue":"komputer-events"}}' > /tmp/config.json

# Run
KOMPUTER_CONFIG_PATH=/tmp/config.json \
KOMPUTER_INSTRUCTIONS="Say hello" \
KOMPUTER_MODEL=claude-sonnet-4-20250514 \
KOMPUTER_AGENT_NAME=test \
ANTHROPIC_API_KEY=sk-ant-... \
python main.py
```

### Build Docker image

```bash
docker build -t komputer-agent:latest .
```

## Project Structure

```
komputer-agent/
├── main.py           # Entrypoint: FastAPI server + initial task
├── agent.py          # Claude Agent SDK integration
├── server.py         # FastAPI endpoints (/task, /cancel, /status)
├── events.py         # Redis event publisher
├── requirements.txt  # Python dependencies
└── Dockerfile        # Python 3.12 + Node.js + Claude CLI
```
