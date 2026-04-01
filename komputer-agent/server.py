import asyncio
import threading
from typing import Optional

from fastapi import BackgroundTasks, FastAPI, HTTPException
from pydantic import BaseModel

import config as agent_config
import state

app = FastAPI()

_publisher = None
_current_task: Optional[asyncio.Task] = None
_current_loop: Optional[asyncio.AbstractEventLoop] = None


def configure(publisher, model: str):
    global _publisher
    _publisher = publisher


class TaskRequest(BaseModel):
    instructions: str
    model: Optional[str] = None
    lifecycle: Optional[str] = None
    system_prompt: Optional[str] = None


class ConfigRequest(BaseModel):
    model: Optional[str] = None
    lifecycle: Optional[str] = None
    role: Optional[str] = None
    instructions: Optional[str] = None
    templateRef: Optional[str] = None
    secrets: Optional[dict[str, str]] = None
    skills: Optional[dict[str, dict]] = None  # name -> {description, content}


@app.get("/status")
async def get_status():
    return {"busy": state.busy.locked()}


@app.get("/healthz")
async def healthz():
    return {"status": "ok"}


@app.get("/readyz")
async def readyz():
    # Check Redis connectivity via the publisher if available.
    if _publisher and hasattr(_publisher, "ping"):
        if not _publisher.ping():
            from fastapi.responses import JSONResponse
            return JSONResponse(
                status_code=503,
                content={"status": "not ready", "error": "redis unreachable"},
            )
    return {"status": "ready"}


@app.post("/config")
async def apply_config(req: ConfigRequest):
    import os as _os
    import re as _re
    # Handle secrets separately — set as env vars, not written to config file.
    if req.secrets:
        for key, value in req.secrets.items():
            sanitized = _re.sub(r"[^A-Za-z0-9]", "_", key).upper()
            env_key = f"SECRET_{sanitized}"
            _os.environ[env_key] = value

    if req.skills:
        from pathlib import Path
        skills_dir = Path.home() / ".claude" / "skills"
        skills_dir.mkdir(parents=True, exist_ok=True)
        for name, skill in req.skills.items():
            md = f"---\nname: {name}\ndescription: {skill['description']}\n---\n\n{skill['content']}"
            (skills_dir / f"{name}.md").write_text(md)

    updates = {k: v for k, v in req.model_dump(exclude={"secrets", "skills"}).items() if v is not None}
    if updates:
        agent_config.apply(updates)

    if not updates and not req.secrets and not req.skills:
        raise HTTPException(status_code=400, detail="No config fields provided")

    return {"status": "applied", "config": agent_config.load()}


@app.post("/task")
async def create_task(req: TaskRequest, background_tasks: BackgroundTasks):
    if state.busy.locked():
        raise HTTPException(status_code=409, detail="Agent is busy with another task")

    # Apply any config overrides from the task request before starting.
    config_updates = {k: v for k, v in {"model": req.model, "lifecycle": req.lifecycle}.items() if v is not None}
    if config_updates:
        agent_config.apply(config_updates)

    from agent import run_agent

    cfg = agent_config.load()
    task_model = cfg["model"]

    def run_with_lock():
        global _current_task, _current_loop
        with state.busy:
            loop = asyncio.new_event_loop()
            state.active_loop = loop
            _current_loop = loop
            _current_task = loop.create_task(run_agent(req.instructions, task_model, _publisher, system_prompt=req.system_prompt))
            try:
                loop.run_until_complete(_current_task)
            except asyncio.CancelledError:
                _publisher.publish("task_cancelled", {"reason": "Cancelled by user"})
            finally:
                state.set_active_client(None)
                state.active_loop = None
                _current_task = None
                _current_loop = None
                loop.close()

    background_tasks.add_task(run_with_lock)
    return {"status": "accepted", "instructions": req.instructions[:100], "model": task_model}


def _interrupt_agent():
    """Interrupt the Claude SDK client and cancel the asyncio task."""
    # First: tell the Claude SDK subprocess to stop gracefully
    if state.active_client and state.active_loop and not state.active_loop.is_closed():
        async def _do_interrupt():
            try:
                await state.active_client.interrupt()
            except Exception:
                pass
        state.active_loop.call_soon_threadsafe(
            lambda: asyncio.ensure_future(_do_interrupt())
        )
    # Then: cancel the Python task to unblock the event loop
    if _current_task and _current_loop and not _current_task.done():
        _current_loop.call_soon_threadsafe(_current_task.cancel)


@app.post("/cancel")
async def cancel_task():
    if not state.busy.locked():
        raise HTTPException(status_code=409, detail="No task is currently running")

    _interrupt_agent()
    return {"status": "cancelling"}


@app.post("/shutdown")
async def shutdown():
    """PreStop hook: wait for the running task to finish, cancel if it takes too long."""
    if not state.busy.locked():
        return {"status": "idle"}

    import time

    # Phase 1: Give the agent up to 20s to finish naturally
    for _ in range(40):
        if not state.busy.locked():
            return {"status": "completed"}
        time.sleep(0.5)

    # Phase 2: Task didn't finish — interrupt via SDK then cancel
    _interrupt_agent()

    # Phase 3: Wait up to 5s for cancellation to flush events
    for _ in range(10):
        if not state.busy.locked():
            return {"status": "cancelled"}
        time.sleep(0.5)

    return {"status": "timeout"}
