# Examples

End-to-end examples covering the full range of komputer.ai capabilities. Each example is self-contained with a README, YAML manifests, and/or code.

## Getting started

All examples assume you have:
- A running Kubernetes cluster with komputer.ai installed (see [Installation](../docs/getting-started/installation.md))
- `kubectl` configured and pointing at your cluster
- `komputer` CLI installed and logged in (`komputer login http://localhost:8080`)

## Examples

| # | Example | What it shows |
|---|---------|---------------|
| [01](01-hello-world/) | **Hello World** | Simplest KomputerAgent — create, run, delete |
| [02](02-reusable-coding-agent/) | **Reusable Coding Agent** | Persistent workspace, follow-up tasks, chat mode |
| [03](03-research-with-secrets/) | **Research with Secrets** | Pass API credentials via K8s Secrets → `SECRET_*` env vars |
| [04](04-manager-with-subagents/) | **Manager with Sub-Agents** | Manager orchestrates parallel workers, KomputerOffice |
| [05](05-scheduled-reports/) | **Scheduled Reports** | KomputerSchedule — cron-triggered agents with timezone |
| [06](06-sleeping-agents/) | **Sleeping Agents** | `lifecycle: Sleep` — cost-efficient agents that spin down between tasks |
| [07](07-custom-agent-image/) | **Custom Agent Image** | Extend the base image with system tools, KomputerAgentClusterTemplate |
| [08](08-python-integration/) | **Python Integration** | HTTP + WebSocket client using `httpx` + `websockets` |
| [09](09-slack-bot/) | **Slack Bot** | Slash command → agent → stream response back to Slack |
| [10](10-ci-cd-integration/) | **CI/CD Integration** | GitHub Actions workflow that triggers an agent and waits for completion |
| [11](11-python-sdk-advanced/) | **Advanced Python SDK** | Use every `create_agent()` field — template, model, role, lifecycle, skills, memories, connectors, secrets |

## Concepts covered

- [Agent lifecycle modes](../docs/concepts/agents.md#lifecycle-modes) (Default, Sleep, AutoDelete) — examples 02, 06
- [Secrets injection](../docs/concepts/secrets.md) — example 03
- [Manager/worker roles and offices](../docs/concepts/agents.md#roles) — example 04
- [Scheduled tasks](../docs/concepts/schedules.md) — example 05
- [Custom images and templates](../docs/integration/custom-agent-image.md) — example 07
- [HTTP + WebSocket API](../docs/integration/) — examples 08, 09, 10
