import json
import os
import threading

import uvicorn

from agent import run_agent_sync
from events import EventPublisher
from server import configure, _busy as busy_lock


def load_config():
    config_path = os.getenv("KOMPUTER_CONFIG_PATH", "/etc/komputer/config.json")
    with open(config_path) as f:
        return json.load(f)


def main():
    instructions = os.getenv("KOMPUTER_INSTRUCTIONS", "")
    model = os.getenv("KOMPUTER_MODEL", "claude-sonnet-4-20250514")
    agent_name = os.getenv("KOMPUTER_AGENT_NAME", "unknown")

    config = load_config()
    redis_config = config.get("redis", {})

    publisher = EventPublisher(redis_config, agent_name)
    print(f"komputer-agent {agent_name} starting with model {model}")

    # Configure the FastAPI server
    configure(publisher, model)

    # Run initial task in a background thread (acquires busy lock)
    def run_initial_task():
        with busy_lock:
            try:
                run_agent_sync(instructions, model, publisher)
            except Exception as e:
                print(f"Initial task failed: {e}", flush=True)
                publisher.publish("error", {"error": str(e)})

    if instructions:
        thread = threading.Thread(target=run_initial_task, daemon=True)
        thread.start()

    # Start FastAPI server (blocks)
    uvicorn.run("server:app", host="0.0.0.0", port=8000)


if __name__ == "__main__":
    main()
