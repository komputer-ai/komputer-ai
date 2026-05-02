---
title: Patterns
description: Common integration recipes — fire-and-forget, create-and-stream, reusable agents, multi-namespace.
---

## Pattern 1: Fire and Forget

Create an agent, don't wait for results. Poll later or check events.

```bash
# Create the agent
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{"name": "reporter", "instructions": "Generate the weekly report"}'

# Check later
curl http://localhost:8080/api/v1/agents/reporter
curl http://localhost:8080/api/v1/agents/reporter/events?limit=5
```

## Pattern 2: Create and Stream

Create an agent and immediately connect via WebSocket to stream results.

```python
import requests
import websocket
import json

API = "http://localhost:8080"

# Create the agent
requests.post(f"{API}/api/v1/agents", json={
    "name": "analyst",
    "instructions": "Analyze server logs for anomalies"
})

# Stream events
ws = websocket.WebSocket()
ws.connect(f"ws://localhost:8080/api/v1/agents/analyst/ws")

while True:
    event = json.loads(ws.recv())
    print(f"[{event['type']}] {event.get('payload', {})}")
    if event["type"] in ("task_completed", "error"):
        break

ws.close()
```

## Pattern 3: Reusable Agents

Create an agent once, then send it multiple tasks over time. The agent keeps its workspace (PVC) between tasks.

```bash
# First task
curl -X POST http://localhost:8080/api/v1/agents \
  -d '{"name": "dev-agent", "instructions": "Clone the repo and set up the project"}'

# Wait for completion, then send another task to the same agent
curl -X POST http://localhost:8080/api/v1/agents \
  -d '{"name": "dev-agent", "instructions": "Run the test suite and fix any failures"}'
```

## Pattern 4: Multi-Namespace Isolation

Run isolated agent pools per team or environment.

> The Anthropic API key Secret only needs to exist once, in the namespace komputer-ai was installed into. The operator mirrors it into each agent namespace (`prod-agents`, `staging-agents`, etc.) automatically.

```bash
# Production agents
curl -X POST "http://localhost:8080/api/v1/agents" \
  -d '{"name": "monitor", "instructions": "Check system health", "namespace": "prod-agents"}'

# Staging agents
curl -X POST "http://localhost:8080/api/v1/agents" \
  -d '{"name": "monitor", "instructions": "Check system health", "namespace": "staging-agents"}'

# List per namespace
curl "http://localhost:8080/api/v1/agents?namespace=prod-agents"
```
