# 09 — Slack Bot

A Slack slash command that runs a komputer.ai agent and posts the result back to the channel.

Uses [Slack Bolt for Python](https://slack.dev/bolt-python/) for the Slack integration and the komputer-ai SDK for agent creation and event streaming.

## What it does

1. Receives a `/agent <instructions>` slash command
2. Immediately acknowledges (Bolt handles the 3-second Slack timeout automatically)
3. Creates a komputer.ai agent in a background thread via the SDK
4. Streams events via `watch_agent()` until `task_completed`
5. Posts the final text response back to the Slack channel

## Install dependencies

```bash
pip install slack-bolt komputer-ai-sdk
```

Bolt includes its own HTTP server — no Flask or additional web framework needed.

## Setup

### 1. Create a Slack app

1. Go to [api.slack.com/apps](https://api.slack.com/apps) → Create New App → From scratch
2. Under **Slash Commands**, create `/agent` with request URL `https://your-server.example.com/slack/events`
3. Under **OAuth & Permissions**, add the `chat:write` bot scope
4. Install the app to your workspace
5. Copy the **Bot User OAuth Token** and **Signing Secret**

### 2. Configure and run

```bash
export SLACK_BOT_TOKEN=xoxb-your-token
export SLACK_SIGNING_SECRET=your-signing-secret
export KOMPUTER_API=http://komputer-api:8080   # or http://localhost:8080 for local dev

python app.py
```

### 3. Expose locally for development

```bash
# Use ngrok to tunnel localhost:3001
ngrok http 3001
# Paste the ngrok URL + /slack/events into your slash command settings
```

## Usage

In any Slack channel where the bot is present:

```
/agent Write a Python script that generates Fibonacci numbers
/agent Explain the CAP theorem in 3 bullet points
/agent What are the best practices for Kubernetes RBAC?
```

Bolt immediately responds: *"Running: <instructions>. I'll post the result here when done."*

Once the agent finishes, the response appears in the channel.

## Why Slack Bolt?

Bolt handles the parts you'd otherwise have to wire up manually:

- **Signature verification** — every request is verified against `SLACK_SIGNING_SECRET` automatically
- **3-second ack** — Bolt separates `ack()` (immediate) from the rest of the handler, so you never time out Slack while the agent runs
- **Routing** — `@app.command("/agent")` cleanly maps commands to handlers without manual URL routing
- **Built-in HTTP server** — `app.start(port=3001)` is all you need; no Flask or separate web framework required

## Key concepts

- **`ack()`** — must be called within 3 seconds; Bolt makes this explicit and hard to forget
- **`client.create_agent(lifecycle="AutoDelete")`** — each slash command creates a fresh agent that deletes itself when done
- **Agent naming** — `slack-{user}` reuses the same agent across commands from the same user; each new command re-tasks it (the SDK handles the 409 automatically)
- **`watch_agent()`** — prefetches event history before opening the WebSocket, so no events are missed even if the agent runs fast
- The komputer.ai API URL should be your in-cluster service address in production
