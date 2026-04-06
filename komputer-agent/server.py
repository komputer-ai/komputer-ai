import asyncio
import threading
from typing import Optional

from fastapi import BackgroundTasks, FastAPI, HTTPException
from pydantic import BaseModel

from agent import _write_skills
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
    secrets: Optional[dict[str, str]] = None  # full set of SECRET_*=value env vars
    skills: Optional[dict[str, dict]] = None  # name -> {description, content}
    mcp_servers: Optional[dict[str, dict]] = None  # connector MCP server configs


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

    # Secrets: full replacement set. Set all present, remove any SECRET_* not in the new set.
    if req.secrets is not None:
        new_keys = set()
        for key, value in req.secrets.items():
            _os.environ[key] = value
            new_keys.add(key)
        for env_key in list(_os.environ.keys()):
            if env_key.startswith("SECRET_") and env_key not in new_keys:
                del _os.environ[env_key]

    if req.skills:
        _write_skills(req.skills)

    # MCP servers: update env var so the next task picks up the new config.
    if req.mcp_servers is not None:
        import json as _json
        _os.environ["KOMPUTER_MCP_SERVERS"] = _json.dumps(req.mcp_servers) if req.mcp_servers else ""

    updates = {k: v for k, v in req.model_dump(exclude={"secrets", "skills", "mcp_servers"}).items() if v is not None}
    if updates:
        agent_config.apply(updates)

    if not updates and req.secrets is None and not req.skills and req.mcp_servers is None:
        raise HTTPException(status_code=400, detail="No config fields provided")

    cfg = agent_config.load()
    from agent import _fetch_context_window
    return {"status": "applied", "config": cfg, "context_window": _fetch_context_window(cfg["model"])}


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
    from agent import _fetch_context_window
    return {"status": "accepted", "instructions": req.instructions[:100], "model": task_model, "context_window": _fetch_context_window(task_model)}


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


@app.get("/history")
async def get_history(limit: int = 50, session_id: str = None):
    """Read messages from the Claude session JSONL file and return as agent events."""
    import json as _json
    from pathlib import Path
    import os

    # Use provided session_id (from API/CR status) or fall back to local file.
    if not session_id:
        from agent import _load_session_id
        session_id = _load_session_id()
    if not session_id:
        return {"events": []}

    # Claude stores sessions at {CLAUDE_CONFIG_DIR}/projects/{project-slug}/{session-id}.jsonl
    claude_dir = Path(os.environ.get("CLAUDE_CONFIG_DIR", str(Path.home() / ".claude")))
    # Find the session file — check all project dirs
    session_file = None
    projects_dir = claude_dir / "projects"
    if projects_dir.exists():
        for project_dir in projects_dir.iterdir():
            candidate = project_dir / f"{session_id}.jsonl"
            if candidate.exists():
                session_file = candidate
                break

    if not session_file or not session_file.exists():
        return {"events": []}

    events = []
    try:
        with open(session_file, "r") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    entry = _json.loads(line)
                except _json.JSONDecodeError:
                    continue

                msg = entry.get("message", {})
                role = msg.get("role", "")
                content = msg.get("content", "")
                timestamp = entry.get("timestamp", "")

                # Convert to AgentEvent format
                if role == "user":
                    # User message — either a string or list of content blocks
                    text = content if isinstance(content, str) else ""
                    if isinstance(content, list):
                        text = " ".join(
                            block.get("text", "") for block in content
                            if isinstance(block, dict) and block.get("type") == "text"
                        )
                    text = text.strip()
                    # Skip IDE context messages (file open, selection, etc.)
                    if not text or text.startswith("<ide_") or text.startswith("<system-reminder"):
                        continue
                    # Strip system prompt prefix (joined by \n\n).
                    parts = text.split("\n\n")
                    if len(parts) > 1:
                        text = parts[-1].strip()
                    if not text or text.startswith("<ide_") or text.startswith("<system-reminder"):
                        continue
                    events.append({
                        "type": "user_message",
                        "timestamp": timestamp,
                        "payload": {"content": text},
                    })
                elif role == "assistant":
                    # Assistant message — content is a list of blocks
                    if isinstance(content, list):
                        for block in content:
                            if not isinstance(block, dict):
                                continue
                            btype = block.get("type", "")
                            if btype == "text":
                                events.append({
                                    "type": "text",
                                    "timestamp": timestamp,
                                    "payload": {"content": block.get("text", "")},
                                })
                            elif btype == "thinking":
                                events.append({
                                    "type": "thinking",
                                    "timestamp": timestamp,
                                    "payload": {"content": block.get("thinking", "")},
                                })
                            elif btype == "tool_use":
                                events.append({
                                    "type": "tool_call",
                                    "timestamp": timestamp,
                                    "payload": {
                                        "tool": block.get("name", ""),
                                        "input": block.get("input", {}),
                                    },
                                })
                            elif btype == "tool_result":
                                events.append({
                                    "type": "tool_result",
                                    "timestamp": timestamp,
                                    "payload": {
                                        "tool": block.get("tool_use_id", ""),
                                        "output": str(block.get("content", ""))[:500],
                                    },
                                })
                    elif isinstance(content, str) and content:
                        events.append({
                            "type": "text",
                            "timestamp": timestamp,
                            "payload": {"content": content},
                        })
    except Exception as e:
        return {"events": [], "error": str(e)}

    # Return last N events
    if limit and len(events) > limit:
        events = events[-limit:]

    return {"events": events}


@app.get("/download/{file_path:path}")
async def download_file(file_path: str):
    """Serve a file from /files/ directory."""
    import os
    from fastapi.responses import FileResponse

    FILES_DIR = "/files"
    # Prevent directory traversal.
    safe_path = os.path.normpath(os.path.join(FILES_DIR, file_path))
    if not safe_path.startswith(FILES_DIR + "/") and safe_path != FILES_DIR:
        raise HTTPException(status_code=400, detail="Invalid file path")
    if not os.path.isfile(safe_path):
        raise HTTPException(status_code=404, detail="File not found")
    return FileResponse(safe_path, filename=os.path.basename(safe_path))


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
