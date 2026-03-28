import asyncio
import json
import os
import signal
import threading

import uvicorn

from agent import run_agent
from events import EventPublisher
import state


def load_redis_config():
    """Load Redis config from file, then override with env vars if set."""
    # Base config from file (if it exists).
    config = {}
    config_path = os.getenv("KOMPUTER_CONFIG_PATH", "/etc/komputer/config.json")
    try:
        with open(config_path) as f:
            config = json.load(f).get("redis", {})
    except (FileNotFoundError, json.JSONDecodeError):
        pass

    # Env vars override file values.
    if os.getenv("KOMPUTER_REDIS_ADDRESS"):
        config["address"] = os.getenv("KOMPUTER_REDIS_ADDRESS")
    if os.getenv("KOMPUTER_REDIS_PASSWORD"):
        config["password"] = os.getenv("KOMPUTER_REDIS_PASSWORD")
    if os.getenv("KOMPUTER_REDIS_DB"):
        config["db"] = int(os.getenv("KOMPUTER_REDIS_DB"))
    if os.getenv("KOMPUTER_REDIS_STREAM_PREFIX"):
        config["stream_prefix"] = os.getenv("KOMPUTER_REDIS_STREAM_PREFIX")

    # Defaults for anything still missing.
    config.setdefault("address", "redis:6379")
    config.setdefault("password", "")
    config.setdefault("db", 0)
    config.setdefault("stream_prefix", "komputer-events")

    return config


def _handle_signal(signum, frame):
    """Handle SIGTERM/SIGINT: interrupt the running task and shut down."""
    sig_name = signal.Signals(signum).name
    print(f"komputer-agent received {sig_name}, shutting down gracefully...", flush=True)
    state.shutdown.set()

    # Interrupt the active Claude SDK client if one is running.
    if state.active_client and state.active_loop and not state.active_loop.is_closed():
        state.active_loop.call_soon_threadsafe(
            lambda: asyncio.ensure_future(_interrupt_client())
        )


async def _interrupt_client():
    """Send interrupt to the active Claude SDK client."""
    try:
        if state.active_client:
            await state.active_client.interrupt()
    except Exception as e:
        print(f"Error interrupting client: {e}", flush=True)


def main():
    instructions = os.getenv("KOMPUTER_INSTRUCTIONS", "")
    model = os.getenv("KOMPUTER_MODEL", "claude-sonnet-4-6")
    agent_name = os.getenv("KOMPUTER_AGENT_NAME", "unknown")

    redis_config = load_redis_config()

    publisher = EventPublisher(redis_config, agent_name)
    print(f"komputer-agent {agent_name} starting with model {model}")

    # Import server here to avoid circular imports; configure it.
    from server import configure
    configure(publisher, model)

    # Register signal handlers before starting any work.
    signal.signal(signal.SIGTERM, _handle_signal)
    signal.signal(signal.SIGINT, _handle_signal)

    # Run initial task in a non-daemon thread (allows graceful shutdown).
    def run_initial_task():
        state.active_loop = asyncio.new_event_loop()
        with state.busy:
            try:
                state.active_loop.run_until_complete(
                    run_agent(instructions, model, publisher)
                )
            except asyncio.CancelledError:
                publisher.publish("task_cancelled", {"reason": "Cancelled by signal"})
            except Exception as e:
                print(f"Initial task failed: {e}", flush=True)
                publisher.publish("error", {"error": str(e)})
            finally:
                state.set_active_client(None)
                state.active_loop = None

    thread = None
    if instructions:
        thread = threading.Thread(target=run_initial_task)
        thread.start()

    # Run uvicorn — it handles SIGTERM/SIGINT for its own shutdown.
    uvicorn.run("server:app", host="0.0.0.0", port=8000)

    # After uvicorn exits, wait for the task thread to finish.
    if thread and thread.is_alive():
        print("Waiting for task to finish...", flush=True)
        thread.join(timeout=10)
        if thread.is_alive():
            print("Task did not finish in time, exiting.", flush=True)


if __name__ == "__main__":
    main()
