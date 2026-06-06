"""Shared mutable state for signal handling and task management."""

import asyncio
import threading

# Active Claude SDK client — set by run_agent, read by signal handler.
active_client = None
active_loop = None

# Threading lock for task execution.
busy = threading.Lock()

# Signal that shutdown has been requested.
shutdown = threading.Event()

# Asyncio queue for steer (follow-up) messages — set by run_agent, cleared when done.
# Messages pushed here are yielded to the SDK's streaming input generator in order.
steer_queue: asyncio.Queue | None = None

# Set to True when an interrupt/cancel is requested — run_agent checks this
# to decide whether to publish task_cancelled vs task_completed.
interrupted = False

# Set to True when a steer interrupts the current task — run_agent skips
# task_cancelled/task_completed and proceeds directly to the steer message.
steered = False

# Seed for the PreCompact hook's tokens_before payload, used when the agent
# spawns a fresh /compact-only task and has no in-session AssistantMessage.usage
# to read yet. The /compact HTTP endpoint sets this from the request body's
# prev_tokens value (the API populates it from CR.status.totalTokens). Cleared
# after the hook reads it.
compaction_prev_tokens: int | None = None


def set_active_client(client):
    """Register or clear the active SDK client."""
    global active_client
    active_client = client


def push_steer_message(message: str) -> bool:
    """Push a steer message onto the queue from a non-async thread.

    Returns True if queued successfully, False if no active session to steer.
    Thread-safe: uses run_coroutine_threadsafe so the put happens inside
    the agent's event loop.
    """
    if steer_queue is None or active_loop is None or active_loop.is_closed():
        return False
    future = asyncio.run_coroutine_threadsafe(steer_queue.put(message), active_loop)
    future.result(timeout=5.0)
    return True
