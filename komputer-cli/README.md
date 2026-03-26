# komputer-cli

Beautiful command-line interface for the komputer.ai platform. Manage agents, stream live events, and monitor task execution — all from your terminal.

Built with [Cobra](https://cobra.dev/) + [Lipgloss](https://github.com/charmbracelet/lipgloss) for colored output and [gorilla/websocket](https://github.com/gorilla/websocket) for real-time streaming.

## Install

```bash
go build -o komputer .
# Move to PATH
mv komputer /usr/local/bin/
```

## Usage

### Login

Save your API endpoint so you don't need to pass it every time:

```bash
komputer login http://localhost:8080
```

Config is stored at `~/.komputer-ai/config.json`. You can override it per-command with `--api`:

```bash
komputer list --api http://other-cluster:8080
```

### Create an agent

```bash
komputer create my-agent "Research the latest AI news and summarize it"
```

With a specific model:

```bash
komputer create my-agent "Write a detailed analysis" --model opus
```

### Run (create + stream)

Creates the agent and streams all events until the task completes:

```bash
komputer run my-agent "Write a haiku about Kubernetes"
```

Output:
```
✔ Agent created
  Streaming events for my-agent...

2026-03-26T10:00:01Z ▶ Task Started
  Write a haiku about Kubernetes

2026-03-26T10:00:03Z 🧠 Thinking
  The user wants a haiku about Kubernetes...

2026-03-26T10:00:03Z 💬 Text
  Pods dance in the cloud
  Orchestrating containers
  Scaling endlessly

2026-03-26T10:00:04Z ✔ Task Completed
  Pods dance in the cloud / Orchestrating containers / Scaling endlessly
  Cost: $0.0039  Duration: 2.9s  Turns: 1
```

### List agents

```bash
komputer list
```

Output:
```
  2 agent(s)

  NAME              PHASE      TASK      MODEL                       CREATED
  ─────────────────────────────────────────────────────────────────────────────
  my-agent          Running    ● Busy    claude-sonnet-4-20250514    2026-03-26T...
  other-agent       Running    ○ Idle    claude-sonnet-4-20250514    2026-03-26T...
```

### Get agent details

```bash
komputer get my-agent
```

### Watch live events

Stream events from an agent via WebSocket (stays connected until Ctrl+C):

```bash
komputer watch my-agent
```

### Cancel a task

```bash
komputer cancel my-agent
```

### Delete an agent

Deletes the agent CR — the operator cleans up the pod, PVC, and ConfigMap:

```bash
komputer delete my-agent
# or
komputer rm my-agent
```

## All Commands

```
komputer login <endpoint>           Save API endpoint
komputer create <name> <prompt>     Create agent or send task [--model, --template]
komputer run <name> <prompt>        Create + stream output    [--model]
komputer list                       List all agents           (alias: ls)
komputer get <name>                 Get agent details
komputer watch <name>               Stream live events (WS)
komputer cancel <name>              Cancel running task
komputer delete <name>              Delete agent              (alias: rm)
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--api <url>` | Override the saved API endpoint |
| `--help` | Help for any command |

## Project Structure

```
komputer-cli/
├── main.go    # All commands, styles, and helpers in one file
├── go.mod
└── go.sum
```
