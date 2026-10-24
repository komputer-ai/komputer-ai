---
title: Agent remote-write
description: Push agent-only metrics to Prometheus reliably even as pods sleep and disappear.
---

Three metrics live only in the agent process — they describe activity that doesn't appear in the Redis event stream:

- `komputer_agent_steering_total` — count of follow-up messages mid-task
- `komputer_agent_mcp_connector_status` — health of each MCP connector (Slack, GitHub, etc.)
- `komputer_agent_subagent_wait_seconds` — wall-clock time spent in `wait_for_agents.py`

When agents sleep (the common case), their `/metrics` endpoint disappears before Prometheus's next scrape. To capture these reliably, configure the agent to push them via Prometheus remote-write:

```yaml
metrics:
  agent:
    remoteWrite:
      enabled: true
      url: http://prometheus.monitoring.svc.cluster.local:9090/api/v1/write
      intervalSeconds: 15
      # Optional bearer token (from a K8s Secret with key 'token'):
      bearerTokenSecret: prometheus-remote-write-token
```

The chart injects `KOMPUTER_METRICS_REMOTE_WRITE_URL`, `KOMPUTER_METRICS_REMOTE_WRITE_INTERVAL`, optionally `KOMPUTER_METRICS_REMOTE_WRITE_TOKEN`, and (when `metrics.perAgentLabels=true`) `KOMPUTER_METRICS_PER_AGENT=true` into the default `KomputerAgentClusterTemplate`, so every agent pod inherits them automatically.

Failures during flush are logged but never crash the agent.

Your remote-write target must have receiver enabled. For Prometheus: `--web.enable-remote-write-receiver`. For Mimir/Cortex/Thanos: native support.

For local kind dev, the sample `KomputerAgentClusterTemplate` already wires the agent to the local Prometheus from `monitoring/docker-compose.yml`. See [local development](../../contribution/local-development.md) and the [local monitoring stack](./local-stack.md).
