# komputer-ai SDK

Official client libraries for the komputer.ai REST API and WebSocket streaming.

## Python

```bash
pip install komputer-ai-sdk
```

```python
from komputer_ai.client import KomputerClient

client = KomputerClient("http://localhost:8080")

# Create an agent
client.create_agent(
    name="my-agent",
    instructions="Analyze our Kubernetes cluster",
    model="claude-sonnet-4-6",
)

# Stream events as the agent works
for event in client.watch_agent("my-agent"):
    if event.type == "text":
        print(event.payload.content)
    elif event.type == "task_completed":
        break

# Attach a memory
client.create_memory(name="context", content="We run a 50-node GKE cluster.")
client.patch_agent("my-agent", memories=["context"])
```

## TypeScript

```bash
npm install @komputer-ai/sdk
```

```typescript
import { KomputerClient } from "@komputer-ai/sdk";

const client = new KomputerClient("http://localhost:8080");

// Create an agent
const agent = await client.createAgent({
  name: "my-agent",
  instructions: "Analyze our Kubernetes cluster",
  model: "claude-sonnet-4-6",
});

// Stream events
for await (const event of client.watchAgent("my-agent")) {
  if (event.type === "text") console.log(event.payload.content);
  if (event.type === "task_completed") break;
}

// Attach a memory
await client.createMemory({ name: "context", content: "We run a 50-node GKE cluster." });
await client.patchAgent({ name: "my-agent", memories: ["context"] });
```

## Go

```bash
go get github.com/komputer-ai/komputer-ai/komputer-sdk/go
```

```go
package main

import (
    "context"
    "fmt"

    client "github.com/komputer-ai/komputer-ai/komputer-sdk/go"
)

func main() {
    c := client.New("http://localhost:8080")
    ctx := context.Background()

    // Create an agent
    agent, _, _ := c.CreateAgent(ctx, "my-agent", "Analyze our cluster",
        client.CreateAgentOpts{Model: client.PtrString("claude-sonnet-4-6")})
    fmt.Println(agent.Name)

    // Stream events
    stream, _ := c.WatchAgent("my-agent")
    defer stream.Close()
    for {
        event, err := stream.Next()
        if err != nil { break }
        fmt.Println(event.Type, event.Payload)
    }
}
```

## Regenerating

When the API changes, regenerate the SDKs:

```bash
cd komputer-sdk

# Regenerate all SDKs
make all

# Or one at a time
make python
make go
make typescript
```

### Prerequisites

- [swag](https://github.com/swaggo/swag) — `go install github.com/swaggo/swag/cmd/swag@v1.16.6`
- Node.js + npx (for openapi-generator-cli)
- Python 3.10+
- Go 1.23+

## Testing

```bash
cd komputer-sdk

# All unit tests
make test

# Integration tests (requires a running komputer-ai instance)
KOMPUTER_API_URL=http://localhost:8080 make test-integration
```

## Structure

```
komputer-sdk/
├── Makefile
├── generate_client.py
├── openapi.yaml
├── python/
│   ├── komputer_ai/
│   │   ├── client.py
│   │   ├── api/
│   │   │   └── agents_ws.py    # WebSocket streaming
│   │   └── models/
│   └── tests/
├── go/
│   ├── client/
│   │   ├── client.go
│   │   ├── watch.go            # WebSocket streaming
│   │   └── client_test.go
│   └── komputer/
├── typescript/
│   └── src/
│       ├── client.ts
│       ├── watch.ts            # WebSocket streaming
│       ├── client.test.ts
│       ├── apis/
│       └── models/
```
