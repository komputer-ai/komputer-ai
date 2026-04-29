---
title: Cost Tracking
description: See what every agent, squad, and schedule costs you in real time — token-by-token, with full historical breakdown.
---

LLM costs are the operating expense of an agent platform. komputer.ai treats cost as a first-class signal: every agent reports what it spent, the platform aggregates it across squads and schedules, and the dashboard surfaces it at every level — from a single tool call up to a fleet-wide trend over weeks.

You always know what you're spending, where, and why.

![Cost dashboard](/cost-page.png)

## What gets tracked

Each agent reports usage to **komputer-api** as it runs:

- **Per-task cost** — what the most recent task spent, in USD.
- **Total cost** — cumulative spend over the agent's entire lifetime, across every task.
- **Token breakdown** — input, output, and cache tokens (read + creation) by model.
- **Context window utilization** — how full the conversation context is, so you can spot agents headed toward auto-compaction or hard limits.

These live on the `KomputerAgent` CR's status, so they're durable, queryable, and survive restarts of every component.

## Where you see it

**The dashboard** — the cost page surfaces your fleet at a glance: total spend over time, breakdown by agent, by model, by squad, and a per-task drill-down so you can find the exact prompt that ran up the bill.

**Per-agent view** — every agent's detail page shows its current task cost, lifetime cost, and a full task-by-task history with token counts.

**Squads and schedules** — these aggregate costs across all the agents they own. A nightly cron schedule that spawns 50 agents reports a single rolled-up cost; a manager squad shows you what the manager spent vs. what its workers spent.

**The CLI** — `komputer get <agent>` includes the cost column. `kubectl get komputeragents` does too — the cost is a CR status field, so anything that reads CRs sees it.

**The API** — `GET /api/v1/agents/:name/cost` returns a full breakdown for an agent, including per-task token counts and per-model spend. The same data is available over the WebSocket as costs accrue, so external dashboards can subscribe to live cost events.

## Why this matters

A platform that runs autonomous agents at scale needs cost observability that matches the autonomy. komputer.ai is built for environments where:

- A single user can spawn dozens of agents in minutes — you need to know which user, which template, and which task is responsible for spend.
- Agents run unattended on schedules — you need historical trends to catch a misbehaving prompt before it becomes a five-figure surprise.
- Different teams share the same cluster — you need per-namespace and per-squad rollups to chargeback or budget accurately.
- Models, prompts, and tools change weekly — you need to compare week-over-week to know whether a change made things cheaper or more expensive.

Cost data is collected the same way the rest of the platform's state is — on the CR — so it integrates with everything that already reads CRs: GitOps, monitoring, custom controllers, your billing pipeline.

## Under the hood

When the Claude Agent SDK in an agent pod completes a task, it emits a `task_completed` event with the full token breakdown. The agent forwards this through Redis to komputer-api, which computes the USD cost from the model's published rates and patches it onto the agent's CR status. The same pipeline updates squad and schedule aggregates.

Because the source of truth is the CR, restarting komputer-api or losing Redis data never loses cost history — it's persisted in the cluster's etcd alongside the agent itself.
