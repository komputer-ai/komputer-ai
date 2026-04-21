"""
Slack slash command → komputer.ai agent → stream response back to Slack.

Uses Slack Bolt for Python and the komputer-ai SDK.

Setup:
  1. Create a Slack app with a slash command (e.g. /agent) pointing at this server.
  2. Set SLACK_BOT_TOKEN and SLACK_SIGNING_SECRET env vars.
  3. pip install slack-bolt komputer-ai-sdk
  4. Run: python app.py

Usage in Slack:
  /agent Write a poem about distributed systems
"""

import os
import re
import threading

from slack_bolt import App
from komputer_ai.client import KomputerClient

SLACK_BOT_TOKEN      = os.environ["SLACK_BOT_TOKEN"]
SLACK_SIGNING_SECRET = os.environ["SLACK_SIGNING_SECRET"]
KOMPUTER_API         = os.environ.get("KOMPUTER_API", "http://komputer-api:8080")

app = App(token=SLACK_BOT_TOKEN, signing_secret=SLACK_SIGNING_SECRET)


def agent_name_for(user: str) -> str:
    slug = re.sub(r"[^a-z0-9-]", "-", user.lower())[:20].strip("-")
    return f"slack-{slug}"


def run_agent_and_reply(agent_name: str, instructions: str, say):
    with KomputerClient(KOMPUTER_API) as client:
        client.create_agent(name=agent_name, instructions=instructions, lifecycle="AutoDelete")

        text_chunks = []
        for event in client.watch_agent(agent_name):
            if event.type == "text":
                text_chunks.append(event.payload.get("content", ""))
            elif event.type == "task_completed":
                break
            elif event.type == "error":
                say(f"Agent error: {event.payload.get('error')}")
                return

    say("\n".join(text_chunks) or "Task completed with no text output.")


@app.command("/agent")
def handle_agent_command(ack, command, say):
    ack(f"Running: _{command['text']}_\nI'll post the result here when done.")

    instructions = command["text"].strip()
    if not instructions:
        ack("Usage: /agent <your task>")
        return

    threading.Thread(
        target=run_agent_and_reply,
        args=(agent_name_for(command.get("user_name", "user")), instructions, say),
        daemon=True,
    ).start()


if __name__ == "__main__":
    app.start(port=3001)
