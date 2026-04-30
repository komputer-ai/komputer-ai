# 11 — Advanced Python SDK

Create an agent that uses **every field** of `KomputerClient.create_agent()` — namespace, template, model, role, lifecycle, system prompt, skills, memories, connectors, secrets, and squad membership.

A more focused intro to the SDK lives in [08-python-integration](../08-python-integration/). This example is for when you need the full surface.

## What it shows

- All 13 fields of `create_agent()`, with comments on what each one does
- Streaming the agent's events as it runs — text, tool calls, thinking, completion

## Install

```bash
pip install komputer-ai-sdk
```

## Pre-requisites in your cluster

The example references named resources that must already exist in the `research` namespace:

| Kind | Name |
|---|---|
| KomputerSkill | `web-research`, `report-writing` |
| KomputerMemory | `team-glossary` |
| KomputerConnector | `slack-prod`, `jira-prod` |
| Secret (K8s) | `openai-api-key` |
| KomputerAgentTemplate | `research-template` |

Drop or substitute any references your cluster doesn't have — every field is optional except `name` and `instructions`.

## Run it

```bash
python agent.py
```

## API reference

The full set of fields and types is documented in:

- **Live Swagger UI:** `http://localhost:8080/swagger/index.html` on a running `komputer-api` — interactive, with descriptions and "Try it out"
- **OpenAPI spec:** [`komputer-sdk/openapi.yaml`](../../komputer-sdk/openapi.yaml) — the source of truth
- **CRD reference:** [`docs/concepts/agents`](../../docs/concepts/agents.md) — every spec field maps 1:1 to the `KomputerAgent` CR

## See also

- [08 — Python Integration](../08-python-integration/) — the simpler starter version
- [Integration overview](../../docs/integration/overview.md)
- [komputer-sdk/python/](../../komputer-sdk/python/) — full SDK reference
