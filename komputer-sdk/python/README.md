# komputer-ai Python SDK

Python client for the [komputer.ai](https://github.com/komputer-ai/komputer-ai) platform.

## Installation

```bash
pip install komputer-ai-sdk
```

Or install directly from the repository:

```bash
pip install git+https://github.com/komputer-ai/komputer-ai.git#subdirectory=komputer-sdk/python
```

## Quick Start

```python
from komputer_ai.client import KomputerClient

client = KomputerClient("http://localhost:8080")

# Create an agent
agent = client.create_agent(
    name="my-agent",
    instructions="Summarize the latest Kubernetes release notes",
    model="claude-sonnet-4-6",
)

# Stream events as the agent works
for event in client.watch_agent("my-agent"):
    if event.type == "text":
        print(event.payload.content)
    elif event.type == "task_completed":
        print(f"Done — cost: ${event.payload.cost_usd}")
        break
```


## Usage

### Agents

```python
# Create
client.create_agent(name="researcher", instructions="Research AI trends", model="claude-sonnet-4-6")

# List
agents = client.list_agents()

# Get
agent = client.get_agent("researcher")

# Update
client.patch_agent("researcher", model="claude-haiku-4-5-20251001", lifecycle="Sleep")

# Send a follow-up task to an idle agent
client.create_agent(name="researcher", instructions="Now focus on LLM benchmarks")

# Cancel a running task
client.cancel_agent_task("researcher")

# Delete
client.delete_agent("researcher")
```

### Memories

```python
client.create_memory(name="company-context", content="We are a B2B SaaS company.", description="Background info")
client.patch_agent("my-agent", memories=["company-context"])

memories = client.list_memories()
client.patch_memory("company-context", content="Updated context.")
client.delete_memory("company-context")
```

### Skills

```python
client.create_skill(name="healthcheck", description="Check service health", content="curl -s http://api/healthz")
client.patch_agent("my-agent", skills=["healthcheck"])

skills = client.list_skills()
client.delete_skill("healthcheck")
```

### Schedules

```python
client.create_schedule(
    name="daily-report",
    schedule="0 9 * * *",
    instructions="Generate a daily status report",
    timezone="America/New_York",
)

schedules = client.list_schedules()
client.patch_schedule("daily-report", schedule="0 10 * * *")
client.delete_schedule("daily-report")
```

### Secrets

```python
client.create_secret(name="api-keys", data={"GITHUB_TOKEN": "ghp_xxx", "SLACK_TOKEN": "xoxb-xxx"})
client.patch_agent("my-agent", secret_refs=["api-keys"])

secrets = client.list_secrets()
client.update_secret("api-keys", data={"GITHUB_TOKEN": "ghp_new"})
client.delete_secret("api-keys")
```

### Connectors

```python
client.create_connector(name="slack", service="slack", url="https://mcp.slack.com", auth_type="token")
client.patch_agent("my-agent", connectors=["slack"])

connectors = client.list_connectors()
client.delete_connector("slack")
```

### Offices

```python
offices = client.list_offices()
office = client.get_office("my-office")
```

### Streaming Events

```python
for event in client.watch_agent("my-agent"):
    match event.type:
        case "task_started":
            print("Agent started working...")
        case "text":
            print(event.payload.content)
        case "tool_call":
            print(f"Using tool: {event.payload.tool}")
        case "task_completed":
            print(f"Done — cost: ${event.payload.cost_usd}")
            break
        case "error":
            print(f"Error: {event.payload.error}")
            break
```

Event types: `task_started`, `thinking`, `tool_use`, `tool_result`, `text`, `task_completed`, `task_cancelled`, `error`.

## Context Manager

```python
with KomputerClient("http://localhost:8080") as client:
    agents = client.list_agents()
```
