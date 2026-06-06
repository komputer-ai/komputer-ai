---
title: Schedules
description: Run agent tasks on a cron schedule.
---

A **KomputerSchedule** runs agent tasks on a cron schedule. Use it for recurring work — nightly reports, periodic monitoring, scheduled analysis.

Key features:

- **Cron expression** — Standard 5-field cron (`min hour dom month dow`)
- **Instructions** — The task prompt the agent runs every tick (editable after creation)
- **Timezone** — IANA timezone support (defaults to UTC)
- **Suspend/resume** — Pause schedules without deleting them
- **Auto-delete** — Optionally delete the schedule after the first successful run
- **Keep agents** — When auto-deleting, optionally keep the created agents alive
- **Agent configuration** — Specify model, role, lifecycle, template, and secrets for created agents
- **Cost tracking** — Tracks total cost and per-run cost across all scheduled runs
- **Manual trigger** — Fire a schedule immediately, outside its cron cadence (UI: "Run now"; CLI: `komputer schedule trigger <name>`)

Schedules default to `Sleep` lifecycle for their agents, so compute is only used during the actual task execution.

## Editing a schedule

Both the cron expression and the instructions can be updated after creation. In the UI, the schedule detail page has inline edit controls for both. From the CLI:

```bash
komputer schedule update my-schedule --cron "0 9 * * 1-5"
komputer schedule update my-schedule --instructions "Summarize yesterday's signups."
```

## Manual trigger

You can fire a schedule outside its normal cron cadence. The schedule's next cron run is unaffected — this is purely an extra one-off run.

```bash
komputer schedule trigger my-schedule
# or
curl -X POST http://komputer-api/api/v1/schedules/my-schedule/trigger
```

The trigger returns `409 Conflict` if the schedule's last run is still in progress.

## Failure behavior

If a scheduled run fails (the agent returns an error), the schedule keeps firing at the next cron tick — failures do not pause the schedule. Only three things stop future runs:

- The schedule is suspended (`spec.suspended: true`)
- The schedule is `autoDelete: true` and has completed (it deletes itself)
- The cron expression is invalid (phase becomes `Error`)

The `failedRuns` counter in the status surfaces past failures so you can spot a repeatedly-failing schedule without it ever blocking itself.
