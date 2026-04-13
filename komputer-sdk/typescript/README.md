# komputer-ai TypeScript SDK

TypeScript/JavaScript client for the [komputer.ai](https://github.com/kontroloop-ai/komputer-ai) platform.

## Installation

```bash
npm install @komputer-ai/sdk
```

## Quick Start

```typescript
import { KomputerClient } from "@komputer-ai/sdk";

const client = new KomputerClient("http://localhost:8080");

// Create an agent
const agent = await client.createAgent({
  name: "my-agent",
  instructions: "Summarize the latest Kubernetes release notes",
  model: "claude-sonnet-4-6",
});

// Stream events as the agent works
for await (const event of client.watchAgent("my-agent")) {
  if (event.type === "text") {
    console.log(event.payload.content);
  } else if (event.type === "task_completed") {
    console.log(`Done — cost: $${event.payload.cost_usd}`);
    break;
  }
}
```

## Usage

### Agents

```typescript
// Create
await client.createAgent({ name: "researcher", instructions: "Research AI trends", model: "claude-sonnet-4-6" });

// List
const agents = await client.listAgents();

// Get
const agent = await client.getAgent("researcher");

// Update
await client.patchAgent({ name: "researcher", model: "claude-haiku-4-5-20251001", lifecycle: "Sleep" });

// Cancel a running task
await client.cancelAgentTask("researcher");

// Delete
await client.deleteAgent("researcher");
```

### Memories

```typescript
await client.createMemory({ name: "company-context", content: "We are a B2B SaaS company.", description: "Background" });
await client.patchAgent({ name: "my-agent", memories: ["company-context"] });

const memories = await client.listMemories();
await client.patchMemory({ name: "company-context", content: "Updated context." });
await client.deleteMemory("company-context");
```

### Skills

```typescript
await client.createSkill({ name: "healthcheck", description: "Check service health", content: "curl -s http://api/healthz" });
await client.patchAgent({ name: "my-agent", skills: ["healthcheck"] });

const skills = await client.listSkills();
await client.deleteSkill("healthcheck");
```

### Schedules

```typescript
await client.createSchedule({
  name: "daily-report",
  schedule: "0 9 * * *",
  instructions: "Generate a daily status report",
  timezone: "America/New_York",
});

const schedules = await client.listSchedules();
await client.patchSchedule({ name: "daily-report", schedule: "0 10 * * *" });
await client.deleteSchedule("daily-report");
```

### Secrets

```typescript
await client.createSecret({ name: "api-keys", data: { GITHUB_TOKEN: "ghp_xxx", SLACK_TOKEN: "xoxb-xxx" } });
await client.patchAgent({ name: "my-agent", secretRefs: ["api-keys"] });

const secrets = await client.listSecrets();
await client.deleteSecret("api-keys");
```

### Connectors

```typescript
await client.createConnector({ name: "slack", service: "slack", url: "https://mcp.slack.com", authType: "token" });
await client.patchAgent({ name: "my-agent", connectors: ["slack"] });

const connectors = await client.listConnectors();
await client.deleteConnector("slack");
```

### Streaming Events

```typescript
for await (const event of client.watchAgent("my-agent")) {
  switch (event.type) {
    case "task_started":
      console.log("Agent started working...");
      break;
    case "text":
      console.log(event.payload.content);
      break;
    case "tool_use":
      console.log(`Using tool: ${event.payload.name}`);
      break;
    case "task_completed":
      console.log(`Done — cost: $${event.payload.cost_usd}`);
      break;
    case "error":
      console.error(event.payload.error);
      break;
  }
}
```

Event types: `task_started`, `thinking`, `tool_use`, `tool_result`, `text`, `task_completed`, `task_cancelled`, `error`.
