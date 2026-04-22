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

Events are published to a per-agent Redis Stream as JSON:

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
| `KOMPUTER_MODEL` | Claude model (e.g. `claude-sonnet-4-6`) |
| `KOMPUTER_AGENT_NAME` | Agent identifier for events |
| `KOMPUTER_NAMESPACE` | Kubernetes namespace (included in events) |
| `KOMPUTER_ROLE` | `manager` or `worker` — managers get MCP orchestration tools |
| `KOMPUTER_API_URL` | Internal komputer-api URL (used by managers for sub-agent management) |
| `ANTHROPIC_API_KEY` | Anthropic API key (from template env/secret) |
| `SECRET_*` | Agent-specific secrets (e.g. `SECRET_GITHUB`, `SECRET_SLACK`) — injected from K8s Secrets listed in `spec.secrets` |

### Secrets

Agents can receive arbitrary secrets via `SECRET_*` env vars. These are created by the API when secrets are passed at agent creation time (e.g. `--secret GITHUB=ghp_xxx` in the CLI). The agent's system prompt instructs it to check these env vars when credentials are needed for a task.

### Config File

Mounted by the operator at `/etc/komputer/config.json`:

```json
{
  "redis": {
    "address": "redis:6379",
    "password": "",
    "db": 0,
    "stream_prefix": "komputer-events"
  }
}
```

## Claude Agent SDK

Uses the [Claude Agent SDK](https://pypi.org/project/claude-agent-sdk/) which wraps the Claude Code CLI. The agent is configured with:

- **Tools:** `Bash`, `WebSearch`
- **Permission mode:** `bypassPermissions` (requires non-root user)
- **Working directory:** `/workspace` (persistent via PVC)

The SDK requires the `claude` CLI binary, which is installed via `npm install -g @anthropic-ai/claude-code` in the Dockerfile.

### Manager Agents

When `KOMPUTER_ROLE=manager`, the agent additionally registers MCP orchestration tools via the komputer-api:

**Agent management:**

| Tool | Description |
|------|-------------|
| `create_agent` | Create a sub-agent (always a worker) to handle a task. Optional `priority` (int, higher = admitted first under template `maxConcurrentAgents` cap) |
| `update_agent` | Patch an existing sub-agent: change `model`, `instructions`, `cpu`, `memory`, `storage`, or `image`. Pass an empty string (e.g. `storage=""`) to remove an override and revert to the template default. Changes apply on next pod start |
| `schedule_agent` | Schedule an agent to run on a cron schedule |
| `get_agent_status` | Check the status of a sub-agent |
| `get_agent_events` | Get recent events/results from a sub-agent |
| `delete_agent` | Delete a sub-agent and clean up resources |
| `delete_schedule` | Delete a schedule |

**Memory tools:**

| Tool | Description |
|------|-------------|
| `create_memory` | Create a `KomputerMemory` CR with the given name and content. Pass `attach: true` to also attach it to the current agent immediately. |
| `attach_memory` | Attach an existing `KomputerMemory` to an agent (defaults to the current agent). The memory content will be injected into the agent's system prompt on its next task. |

**Skill tools:**

| Tool | Description |
|------|-------------|
| `create_skill` | Create a `KomputerSkill` CR with the given name, description, and content. Pass `attach: true` to also attach it to the current agent immediately. |
| `attach_skill` | Attach an existing `KomputerSkill` to an agent (defaults to the current agent). The skill will be written as a slash command file on the agent's next task. |

This allows manager agents to autonomously delegate work, build up shared knowledge as memories, and codify repeatable workflows as skills.

## Development

### Prerequisites

- Python 3.12+
- Node.js 22+ (for Claude Code CLI)

### Local setup

```bash
pip install -r requirements.txt
npm install -g @anthropic-ai/claude-code

# Create a config file
echo '{"redis":{"address":"localhost:6379","db":0,"stream_prefix":"komputer-events"}}' > /tmp/config.json

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
├── manager_tools.py  # MCP tools for manager agents (sub-agent orchestration)
├── requirements.txt  # Python dependencies
└── Dockerfile        # Python 3.12 + Node.js + Claude CLI
```
