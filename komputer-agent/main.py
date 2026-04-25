import asyncio
import json
import logging
import os
import signal
import threading

import uvicorn

from agent import run_agent, _write_skills
from events import EventPublisher
from logger import init_logger
import config
import metrics as agent_metrics
import state

logger = logging.getLogger(__name__)


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
    logger.info("received signal, shutting down gracefully", extra={"signal": sig_name})
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
        logger.exception("error interrupting client", extra={"error": str(e)})


def main():
    instructions = os.getenv("KOMPUTER_INSTRUCTIONS", "")
    agent_name = os.getenv("KOMPUTER_AGENT_NAME", "unknown")

    internal_system_prompt = os.getenv("KOMPUTER_INTERNAL_SYSTEM_PROMPT", "")
    user_system_prompt = os.getenv("KOMPUTER_SYSTEM_PROMPT", "")
    combined_prompt_parts = [p for p in [internal_system_prompt, user_system_prompt] if p]
    combined_system_prompt = "\n\n".join(combined_prompt_parts) if combined_prompt_parts else None

    # Initialize structured logger (must happen before anything else logs).
    init_logger()

    # Initialize metrics module (must happen before server starts).
    agent_metrics.init()

    # Initialize runtime config from env vars.
    config.init()

    # Write skill files from SKILL_* env vars
    env_skills = {
        key[6:].lower().replace("_", "-"): value
        for key, value in os.environ.items()
        if key.startswith("SKILL_")
    }
    if env_skills:
        _write_skills(env_skills)

    cfg = config.load()
    model = cfg["model"]

    redis_config = load_redis_config()

    publisher = EventPublisher(redis_config, agent_name)
    logger.info("agent starting", extra={"agent_name": agent_name, "model": model})

    # Import server here to avoid circular imports; configure it.
    from server import configure
    configure(publisher, model)

    # Register signal handlers before starting any work.
    signal.signal(signal.SIGTERM, _handle_signal)
    signal.signal(signal.SIGINT, _handle_signal)

    # Run initial task in a non-daemon thread (allows graceful shutdown).
    # The steer_queue is created inside run_agent and stored in state, so
    # follow-up messages posted to /task while this is running will be queued.
    def run_initial_task():
        state.active_loop = asyncio.new_event_loop()
        with state.busy:
            try:
                state.active_loop.run_until_complete(
                    run_agent(instructions, model, publisher, system_prompt=combined_system_prompt)
                )
            except asyncio.CancelledError:
                publisher.publish("task_cancelled", {"reason": "Cancelled by signal"})
            except Exception as e:
                logger.exception("initial task failed", extra={"error": str(e)})
                publisher.publish("error", {"error": str(e)})
            finally:
                state.set_active_client(None)
                state.active_loop = None
                state.steer_queue = None  # Ensure no stale queue reference remains.

    thread = None
    if instructions:
        thread = threading.Thread(target=run_initial_task)
        thread.start()

    # Run uvicorn — it handles SIGTERM/SIGINT for its own shutdown.
    uvicorn.run("server:app", host="0.0.0.0", port=8000)

    # After uvicorn exits, wait for the task thread to finish.
    if thread and thread.is_alive():
        logger.info("waiting for task to finish")
        thread.join(timeout=10)
        if thread.is_alive():
            logger.warning("task did not finish in time, exiting")

    # Flush any remaining queued events to Redis before exiting.
    publisher.shutdown()


if __name__ == "__main__":
    main()
