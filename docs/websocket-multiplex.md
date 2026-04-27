# Multi-agent WebSocket subscriptions

The `/api/v1/agents/events/ws` endpoint streams events from many agents over a single WebSocket connection. Use it for dashboards, observability pipelines, or any client that would otherwise open one WebSocket per agent.

## Selecting agents

Combine any of these query parameters; results are unioned:

| Param | Example | Behavior |
|---|---|---|
| `match` | `match=worker-*` | Glob pattern on agent name. Comma-separated for multiple patterns. Syntax is Go `path.Match`: `*`, `?`, `[a-z]`. |
| `agents` | `agents=alice,bob` | Explicit comma-separated agent names. |
| `namespace` | `namespace=team-a` | Optional namespace filter. Omit to receive events from any namespace. |

At least one of `match` or `agents` is required.

## Examples

```
WS /api/v1/agents/events/ws?match=*                      # all agents in any namespace
WS /api/v1/agents/events/ws?match=worker-*&namespace=foo # all "worker-*" in namespace foo
WS /api/v1/agents/events/ws?agents=coder,reviewer        # exactly these two
WS /api/v1/agents/events/ws?match=worker-*&agents=mgr-1  # workers + manager-1
```

## Message format

Each message is a JSON-encoded `AgentEvent`:

```json
{
  "agentName": "worker-3",
  "namespace": "default",
  "type": "text",
  "timestamp": "2026-04-27T10:00:00.123Z",
  "payload": { "text": "..." }
}
```

The `agentName` and `namespace` fields let the client demultiplex by source.

## Dynamic membership

When a new agent publishes its first event, if its name matches a connected subscription's `match` pattern, that event and all subsequent events are delivered automatically. No reconnection is needed.

## Limits

- Up to **200 agents** per explicit `agents=` list per connection.
- Wildcards have no hard cap on matched agents.
- Each connection has a **256-message bounded send queue**. If a client falls behind, the oldest queued messages are dropped (and counted in `komputer_api_ws_send_queue_dropped_total{mode="match"}`).

## Comparison with the per-agent endpoint

The per-agent endpoint at `GET /api/v1/agents/:name/ws` is unchanged and still recommended for single-agent UI views. The multi endpoint is preferable when monitoring more than 2-3 agents, since it keeps a single WebSocket connection open instead of N.
