"""Shared mutable state for signal handling and task management."""

import threading

# Active Claude SDK client — set by run_agent, read by signal handler.
active_client = None
active_loop = None

# Threading lock for task execution.
busy = threading.Lock()

# Signal that shutdown has been requested.
shutdown = threading.Event()


def set_active_client(client):
    """Register or clear the active SDK client."""
    global active_client
    active_client = client
