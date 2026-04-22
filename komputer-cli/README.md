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

### Namespace targeting

All commands support namespace selection via `--namespace` (or `-n`):

```bash
komputer -n production list
komputer -n staging run my-agent "Deploy the new feature"
```

If omitted, the server's default namespace is used.

### Create an agent

```bash
komputer create my-agent "Research the latest AI news and summarize it"
```

With a specific model:

```bash
komputer create my-agent "Write a detailed analysis" --model opus
```

With per-agent overrides (resources, image, storage):

```bash
komputer create heavy-job "Process the dataset" \
  --cpu 4 --memory-limit 8Gi --storage 50Gi
```

With queue priority (used when the template has `maxConcurrentAgents > 0`):

```bash
komputer create urgent-fix "Hotfix the deploy" --priority 100
```

### Update an agent

`komputer update` patches an existing agent's spec. Same flags as `create` (model, instructions, priority, cpu, memory-limit, storage, image). Changes apply when the next pod is built — running pods are not mutated:

```bash
komputer update my-agent --priority 50 --memory-limit 4Gi
```

To clear an override and revert to the template default, pass an empty value:

```bash
komputer update my-agent --storage ""        # remove storage override
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

  NAME              PHASE      TASK             MODEL               CREATED
  ──────────────────────────────────────────────────────────────────────────────
  my-agent          Running    ● In Progress    claude-sonnet-4-6   2026-03-26T...
  other-agent       Running    ✔ Complete       claude-sonnet-4-6   2026-03-26T...
```

Filter by phase (useful when a template's `maxConcurrentAgents` cap is in play):

```bash
komputer list --status queued
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

### Chat (interactive)

Start a turn-by-turn conversation with an agent. The agent is auto-created on your first message if it doesn't exist yet. The agent persists between turns, keeping its workspace and conversation history.

```bash
komputer chat my-agent
```

```
  Chat with my-agent
  Type a message and press Enter. Ctrl+C to interrupt or exit.

> What files are in /workspace?
  ⚙ Bash $ ls -la /workspace

my-agent:
The workspace contains the following files...

✔ Cost: $0.0042  Duration: 3.2s

> Now create a Python script that prints hello world
  ⚙ Bash $ cat > /workspace/hello.py << 'EOF' ...

my-agent:
I've created hello.py in your workspace.

✔ Cost: $0.0031  Duration: 2.1s
```

- **Ctrl+C during a response** — cancels the current turn (agent stays alive)
- **Ctrl+C at the prompt** — exits the chat session (agent stays alive)
- Shows text responses and tool summaries, hides thinking and raw events
- Supports `--model` and `--lifecycle` flags

### Delete an agent

Deletes the agent CR — the operator cleans up the pod, PVC, and ConfigMap:

```bash
komputer delete my-agent
# or
komputer rm my-agent
```

## All Commands

### Agents
```
komputer login <endpoint>           Save API endpoint
komputer create <name> <prompt>     Create agent or send task [--model, --template, --lifecycle, --system-prompt, --priority, --cpu, --memory-limit, --storage, --image]
komputer run <name> <prompt>        Create + stream output    [--model, --lifecycle, --system-prompt, --priority, --cpu, --memory-limit, --storage, --image]
komputer update <name>              Patch existing agent       [--model, --instructions, --priority, --cpu, --memory-limit, --storage, --image] (pass empty value to clear an override)
komputer chat <name>                Interactive conversation   [--model, --lifecycle]
komputer list                       List all agents            [--status queued]  (alias: ls)
komputer get <name>                 Get agent details
komputer watch <name>               Stream live events (WS)
komputer cancel <name>              Cancel running task
komputer delete <name> [name...]    Delete one or more agents (alias: rm)
```

### Offices
```
komputer office list                List all offices
komputer office get <name>          Get office details + members
komputer office events <name>       Get office event history
komputer office delete <name>       Delete office and its agents
```

### Schedules
```
komputer schedule list              List all schedules
komputer schedule get <name>        Get schedule details
komputer schedule create <name> <cron> <prompt>  Create a schedule [--model, --lifecycle, --timezone]
komputer schedule delete <name>     Delete a schedule
```

### Memories
```
komputer memory list                List all memories
komputer memory get <name>          Get memory details + content
komputer memory create <name>       Create a memory [--content, --description]
komputer memory edit <name>         Update content or description [--content, --description]
komputer memory delete <name>       Delete a memory
```

### Skills
```
komputer skill list                 List all skills
komputer skill get <name>           Get skill details + content
komputer skill create <name>        Create a skill [--content, --description]
komputer skill edit <name>          Update content or description [--content, --description]
komputer skill delete <name>        Delete a skill
```

### Memories

Manage `KomputerMemory` resources — persistent knowledge injected into agent system prompts.

```bash
# List all memories
komputer memory list

# Get memory details (including content)
komputer memory get k8s-runbook

# Create a memory
komputer memory create k8s-runbook \
  --description "Kubernetes runbook for production cluster" \
  --content "Always drain nodes before maintenance..."

# Edit a memory's content or description
komputer memory edit k8s-runbook --content "Updated content..."

# Delete a memory
komputer memory delete k8s-runbook
```

Attach memories when creating an agent:
```bash
komputer create my-agent "Deploy the app" --memory k8s-runbook --memory deploy-policy
```

Update memories on an existing agent:
```bash
komputer config my-agent --memory k8s-runbook --memory deploy-policy
```

### Skills

Manage `KomputerSkill` resources — reusable skill files available to agents as slash commands.

```bash
# List all skills
komputer skill list

# Get skill details (including content)
komputer skill get python-expert

# Create a skill
komputer skill create python-expert \
  --description "Expert Python code review" \
  --content "When reviewing Python code: check PEP 8, look for security issues..."

# Edit a skill's content or description
komputer skill edit python-expert --content "Updated instructions..."

# Delete a skill
komputer skill delete python-expert
```

Attach skills when creating an agent:
```bash
komputer create my-agent "Review this PR" --skill python-expert --skill git-commit
```

Update skills on an existing agent:
```bash
komputer config my-agent --skill python-expert
```

### Passing Secrets

Pass credentials to agents at creation time using the `--secret` flag:

```bash
# Single secret
komputer run github-bot "create a PR" --secret GITHUB=ghp_xxx

# Multiple secrets
komputer create deploy-agent "deploy to prod" \
  --secret GITHUB=ghp_xxx \
  --secret SLACK=xoxb-xxx \
  --secret AWS_KEY=AKIA...
```

Secrets are stored as K8s Secrets and injected as `SECRET_*` env vars into the agent pod. The agent checks these env vars when credentials are needed.

## JSON Output

All commands that return structured data support `--json` for machine-readable output:

```bash
komputer list --json
komputer get my-agent --json
komputer create my-agent "Do the thing" --json
komputer delete my-agent --json
komputer cancel my-agent --json
komputer config my-agent --model claude-opus-4-6 --json
komputer office list --json
komputer office get my-office --json
komputer schedule list --json
komputer schedule get my-schedule --json
komputer schedule create my-schedule "0 9 * * *" "Daily report" --json
komputer schedule delete my-schedule --json
komputer memory list --json
komputer memory get k8s-runbook --json
komputer memory create k8s-runbook --content "..." --json
komputer memory edit k8s-runbook --content "..." --json
komputer memory delete k8s-runbook --json
komputer skill list --json
komputer skill get python-expert --json
komputer skill create python-expert --content "..." --json
komputer skill edit python-expert --content "..." --json
komputer skill delete python-expert --json
```

In JSON mode:
- Human-formatted output is suppressed; raw JSON is written to stdout
- Errors are written to stderr as `{"error": "...", "status": 404}` and exit with a non-zero code
- `delete` commands return an array: `[{"name": "my-agent", "deleted": true}]`
- `cancel` returns `{"name": "my-agent", "cancelled": true}`
- `get` commands include events inline: `{"agent": {...}, "events": [...]}`
- Streaming commands (`run`, `watch`, `chat`, `office watch`) do not support `--json`

Useful for scripting:
```bash
# Get all agent names
komputer list --json | jq -r '.agents[].name'

# Check if a task completed successfully
komputer get my-agent --json | jq -r '.agent.taskStatus'

# Create and capture the agent name
name=$(komputer create my-agent "Do something" --json | jq -r '.name')
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--api <url>` | Override the saved API endpoint |
| `-n, --namespace <ns>` | Target Kubernetes namespace |
| `--json` | Output raw JSON instead of formatted text |
| `--secret KEY=VALUE` | Pass secrets to the agent (repeatable, on create/run) |
| `--memory <name>` | Attach a KomputerMemory by name (repeatable, on create/run/config) |
| `--skill <name>` | Attach a KomputerSkill by name (repeatable, on create/run/config) |
| `--lifecycle <mode>` | Agent lifecycle: `Sleep` or `AutoDelete` (on create/run/chat) |
| `--system-prompt <text>` | Custom system prompt for the agent (on create/run/config) |
| `--priority <int>` | Queue priority — higher = admitted first when the template's `maxConcurrentAgents` is reached. Default 0 (on create/update) |
| `--cpu <quantity>` | Override agent container CPU, sets both requests and limits (on create/update) |
| `--memory-limit <quantity>` | Override agent container memory, sets both requests and limits (on create/update) |
| `--storage <size>` | Override PVC size; expands existing PVCs in place when StorageClass supports it (on create/update) |
| `--image <image>` | Override the agent container image (on create/update) |
| `--status <phase>` | Filter `list` by phase (e.g. `--status queued`) |
| `--help` | Help for any command |

## Project Structure

```
komputer-cli/
├── main.go            # Root command setup + register calls
├── styles.go          # Lipgloss style definitions
├── spinner.go         # Terminal spinner for loading states
├── config.go          # CLI config (~/.komputer-ai/config.json)
├── types.go           # API response/request structs
├── helpers.go         # Shared helpers (apiRequest, formatEvent, printAgent, etc.)
├── cmd_agents.go      # Agent commands: list, create, get, delete, cancel, config, watch, run, chat, login
├── cmd_offices.go     # Office commands: list, get, watch, delete
├── cmd_schedules.go   # Schedule commands: list, get, create, delete
├── cmd_memories.go    # Memory commands: list, get, create, edit, delete
├── cmd_skills.go      # Skill commands: list, get, create, edit, delete
├── go.mod
└── go.sum
```
