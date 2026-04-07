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
