# Logging

komputer-api and komputer-agent both emit **structured logs** with consistent fields, suitable for ingestion by log aggregators (Loki, Elasticsearch, Datadog, etc.).

## Format

Auto-detected from stdout:

| Context | Format |
|---|---|
| Running in a container or piped (no TTY) | JSON, one object per line |
| Running interactively in a terminal | Colored, human-readable |

Override with `LOG_FORMAT=json|text`.

## Level

Controlled by the `LOG_LEVEL` env var, default `info`. Accepted: `debug`, `info`, `warn`, `error`.

```bash
LOG_LEVEL=debug helm upgrade ...    # see verbose internals
```

## Required fields

Every log line includes:

| Field | Description |
|---|---|
| `timestamp` | ISO 8601 |
| `level` | `debug` / `info` / `warn` / `error` |
| `component` | `komputer-api` or `komputer-agent` |
| `message` | Short human description |

Common contextual fields (when relevant): `agent_name`, `namespace`, `tool_name`, `connector_name`, `error`, `duration_ms`, `session_id`.

## Example log lines

JSON (production):

```json
{"timestamp":"2026-04-25T10:00:00","level":"info","component":"komputer-api","message":"redis worker started"}
{"timestamp":"2026-04-25T10:00:01","level":"error","component":"komputer-agent","message":"failed to publish event","error":"connection refused"}
```

Text (local dev):

```
2026-04-25T10:00:00 INFO  redis worker started
2026-04-25T10:00:01 ERROR failed to publish event  error=connection refused
```

## Quieted libraries

The agent suppresses these noisy library loggers below WARN by default: `httpx`, `httpcore`, `uvicorn.access`. They re-emerge when `LOG_LEVEL=debug`.

## What's NOT structured-logged

- **HTTP access logs** — covered by Prometheus metrics ([monitoring](monitoring.md)) instead. The default gin access logger is disabled.
- **Operator logs** — controller-runtime emits its own format; out of scope for this pass (issue #111 follow-up).
- **CLI output** — `komputer-cli` is a human-facing terminal app, not a structured-log producer.

## Troubleshooting

**Logs aren't JSON in my container?** Check that stdout isn't attached to a TTY (`docker run -it` keeps stdout as a TTY → text format). Force with `LOG_FORMAT=json`.

**Want to grep a specific agent?** With JSON: `kubectl logs <pod> | jq 'select(.agent_name == "foo")'`. With text: not easily — use JSON for filtering.

**Verbose Claude SDK noise?** Set `LOG_LEVEL=warn` to suppress lifecycle Info logs. Or `LOG_LEVEL=error` for failures only.
