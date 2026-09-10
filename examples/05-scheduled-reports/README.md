# 05 — Scheduled Reports

Run an agent on a cron schedule. The agent generates a daily standup report every weekday morning.

## What it does

A `KomputerSchedule` triggers an agent at 9 AM Eastern time every weekday (Monday–Friday). Each run creates a new agent, generates the report, and then the agent auto-deletes itself. The schedule persists indefinitely.

## Run it

```bash
# Apply the schedule
kubectl apply -f schedule.yaml

# Check schedule status
kubectl get komputerschedules
komputer schedule list
komputer schedule get daily-standup-report
```

## Schedule fields explained

```yaml
schedule: "0 9 * * 1-5"       # 9 AM, Monday through Friday
timezone: "America/New_York"  # IANA timezone
agent:
  lifecycle: AutoDelete       # each run creates + deletes a fresh agent
  role: worker                # see note below
  skills: [markdown-reports]  # attach a KomputerSkill
  disallowedTools: [Bash]     # keep every other tool, drop the shell
  storage: { size: 20Gi }     # override the template's PVC size
```

`spec.agent` accepts **every field a `KomputerAgent` spec accepts except `instructions`** — those come from the schedule's own top-level `instructions`. See [Schedules → Agent configuration](../../docs/concepts/schedules.md#agent-configuration) for the full list.

Two things to know about this example:

- **`role: worker` is set explicitly on purpose.** The `worker` default for schedules is applied by the API, CLI, UI, and SDK — not by the CRD. A schedule applied straight through `kubectl` picks up the agent CRD's own default of `manager` unless you say otherwise.
- **The `markdown-reports` skill and `team-conventions` memory must exist**, or they are silently skipped (the operator logs and continues). Create them first, or drop those two lines.

## Managing the schedule

```bash
# Suspend without deleting
kubectl patch komputerschedule daily-standup-report \
  --type=merge -p '{"spec":{"suspended":true}}'

# Resume
kubectl patch komputerschedule daily-standup-report \
  --type=merge -p '{"spec":{"suspended":false}}'

# Delete the schedule (does not affect agents already running)
kubectl delete komputerschedule daily-standup-report
komputer schedule delete daily-standup-report
```

## Targeting an existing agent

Instead of creating a new agent each run, you can send tasks to an existing sleeping agent:

```yaml
spec:
  schedule: "0 9 * * 1-5"
  instructions: "Generate the daily report..."
  agentName: my-persistent-agent   # wake this agent instead of creating a new one
```

The agent must have `lifecycle: Sleep` so it's available to be woken.

## Key concepts

- **`KomputerSchedule`** — cron-triggered agent tasks, managed by the operator
- **`timezone`** — IANA timezone string; defaults to UTC if omitted
- **`autoDelete: true`** — delete the schedule CR after the first successful run (one-shot jobs)
- **`suspended: true`** — pause the schedule without losing it
- Cost per run is tracked in `status.lastRunCostUSD` and `status.totalCostUSD`
