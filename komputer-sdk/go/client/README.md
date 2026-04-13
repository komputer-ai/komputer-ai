# komputer-ai Go SDK

Go client for the [komputer.ai](https://github.com/kontroloop-ai/komputer-ai) platform.

## Installation

```bash
go get github.com/kontroloop-ai/komputer-ai/komputer-sdk/go/client
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    client "github.com/kontroloop-ai/komputer-ai/komputer-sdk/go/client"
    komputer "github.com/kontroloop-ai/komputer-ai/komputer-sdk/go/komputer"
)

func main() {
    c := client.New("http://localhost:8080")
    ctx := context.Background()

    // Create an agent
    agent, _, err := c.CreateAgent(ctx, "my-agent", "Summarize the latest Kubernetes release notes",
        client.CreateAgentOpts{Model: komputer.PtrString("claude-sonnet-4-6")})
    if err != nil {
        panic(err)
    }
    fmt.Println("Created:", agent.Name)

    // Stream events as the agent works
    stream, err := c.WatchAgent("my-agent")
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
    client.CreateAgentOpts{Model: komputer.PtrString("claude-sonnet-4-6")})

// List
agents, _, _ := c.ListAgents(ctx)

// Get
agent, _, _ := c.GetAgent(ctx, "researcher")

// Update
c.PatchAgent(ctx, "researcher",
    client.PatchAgentOpts{
        Model:     komputer.PtrString("claude-haiku-4-5-20251001"),
        Lifecycle: komputer.PtrString("Sleep"),
    })

// Cancel a running task
c.CancelAgentTask(ctx, "researcher")

// Delete
c.DeleteAgent(ctx, "researcher")
```

### Memories

```go
c.CreateMemory(ctx, "company-context", "We are a B2B SaaS company.",
    client.CreateMemoryOpts{Description: komputer.PtrString("Background info")})
c.PatchAgent(ctx, "my-agent", client.PatchAgentOpts{Memories: []string{"company-context"}})

c.PatchMemory(ctx, "company-context",
    client.PatchMemoryOpts{Content: komputer.PtrString("Updated context.")})
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
    client.CreateScheduleOpts{Timezone: komputer.PtrString("America/New_York")})

c.PatchSchedule(ctx, "daily-report",
    client.PatchScheduleOpts{Schedule: komputer.PtrString("0 10 * * *")})
c.DeleteSchedule(ctx, "daily-report")
```

### Secrets

```go
c.CreateSecret(ctx, map[string]string{"GITHUB_TOKEN": "ghp_xxx"}, "api-keys")
c.PatchAgent(ctx, "my-agent", client.PatchAgentOpts{SecretRefs: []string{"api-keys"}})
c.DeleteSecret(ctx, "api-keys")
```

### Connectors

```go
c.CreateConnector(ctx, "slack", "slack", "https://mcp.slack.com",
    client.CreateConnectorOpts{AuthType: komputer.PtrString("token")})
c.PatchAgent(ctx, "my-agent", client.PatchAgentOpts{Connectors: []string{"slack"}})
c.DeleteConnector(ctx, "slack")
```

### Streaming Events

```go
stream, err := c.WatchAgent("my-agent")
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
