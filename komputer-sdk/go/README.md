# komputer-ai Go SDK

Go client for the [komputer.ai](https://github.com/komputer-ai/komputer-ai) platform.

## Installation

```bash
go get github.com/komputer-ai/komputer-ai/komputer-sdk/go
```

## Quick Start

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
    agent, _, err := c.CreateAgent(ctx, "my-agent", "Summarize the latest Kubernetes release notes",
        client.CreateAgentOpts{Model: client.PtrString("claude-sonnet-4-6")})
    if err != nil {
        panic(err)
    }
    fmt.Println("Created:", agent.Name)

    // Stream events as the agent works
    stream, err := c.WatchAgent(ctx, "my-agent")
    if err != nil {
        panic(err)
    }
    defer stream.Close()
    for {
        event, err := stream.Next()
        if err != nil {
            break
        }
        switch event.Type {
        case "text":
            fmt.Println(event.Payload["content"])
        case "task_completed":
            fmt.Printf("Done — cost: $%v\n", event.Payload["cost_usd"])
            return
        }
    }
}
```

## Usage

### Agents

```go
ctx := context.Background()

// Create
agent, _, _ := c.CreateAgent(ctx, "researcher", "Research AI trends",
    client.CreateAgentOpts{Model: client.PtrString("claude-sonnet-4-6")})

// List
agents, _, _ := c.ListAgents(ctx)

// Get
agent, _, _ := c.GetAgent(ctx, "researcher")

// Update
c.PatchAgent(ctx, "researcher",
    client.PatchAgentOpts{
        Model:     client.PtrString("claude-haiku-4-5-20251001"),
        Lifecycle: client.PtrString("Sleep"),
    })

// Cancel a running task
c.CancelAgentTask(ctx, "researcher")

// Delete
c.DeleteAgent(ctx, "researcher")
```

### Memories

```go
c.CreateMemory(ctx, "company-context", "We are a B2B SaaS company.",
    client.CreateMemoryOpts{Description: client.PtrString("Background info")})
c.PatchAgent(ctx, "my-agent", client.PatchAgentOpts{Memories: []string{"company-context"}})

c.PatchMemory(ctx, "company-context",
    client.PatchMemoryOpts{Content: client.PtrString("Updated context.")})
c.DeleteMemory(ctx, "company-context")
```

### Skills

```go
c.CreateSkill(ctx, "healthcheck", "curl -s http://api/healthz", "Check service health")
c.PatchAgent(ctx, "my-agent", client.PatchAgentOpts{Skills: []string{"healthcheck"}})
c.DeleteSkill(ctx, "healthcheck")
```

### Schedules

```go
c.CreateSchedule(ctx, "daily-report", "Generate a daily status report", "0 9 * * *",
    client.CreateScheduleOpts{Timezone: client.PtrString("America/New_York")})

c.PatchSchedule(ctx, "daily-report",
    client.PatchScheduleOpts{Schedule: client.PtrString("0 10 * * *")})
c.DeleteSchedule(ctx, "daily-report")
```

### Secrets

```go
c.CreateSecret(ctx, "api-keys", map[string]string{"GITHUB_TOKEN": "ghp_xxx"})
c.PatchAgent(ctx, "my-agent", client.PatchAgentOpts{SecretRefs: []string{"api-keys"}})
c.DeleteSecret(ctx, "api-keys")
```

### Connectors

```go
c.CreateConnector(ctx, "slack", "slack", "https://mcp.slack.com",
    client.CreateConnectorOpts{AuthType: client.PtrString("token")})
c.PatchAgent(ctx, "my-agent", client.PatchAgentOpts{Connectors: []string{"slack"}})
c.DeleteConnector(ctx, "slack")
```

### Streaming Events

```go
stream, err := c.WatchAgent(ctx, "my-agent")
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for {
    event, err := stream.Next()
    if err != nil {
        break
    }
    switch event.Type {
    case "task_started":
        fmt.Println("Agent started working...")
    case "text":
        fmt.Println(event.Payload["content"])
    case "tool_use":
        fmt.Printf("Using tool: %s\n", event.Payload["name"])
    case "task_completed":
        fmt.Printf("Done — cost: $%v\n", event.Payload["cost_usd"])
    case "error":
        fmt.Printf("Error: %v\n", event.Payload["error"])
    }
}
```

Event types: `task_started`, `thinking`, `tool_use`, `tool_result`, `text`, `task_completed`, `task_cancelled`, `error`.
