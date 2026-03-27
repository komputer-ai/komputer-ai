import asyncio
import threading
from typing import Optional

from fastapi import BackgroundTasks, FastAPI, HTTPException
from pydantic import BaseModel

import state

app = FastAPI()

_publisher = None
_model = None
_current_task: Optional[asyncio.Task] = None
_current_loop: Optional[asyncio.AbstractEventLoop] = None


def configure(publisher, model: str):
    global _publisher, _model
    _publisher = publisher
    _model = model


class TaskRequest(BaseModel):
    instructions: str
    model: Optional[str] = None


@app.get("/status")
async def get_status():
    return {"busy": state.busy.locked()}


@app.post("/task")
async def create_task(req: TaskRequest, background_tasks: BackgroundTasks):
    if state.busy.locked():
        raise HTTPException(status_code=409, detail="Agent is busy with another task")

    from agent import run_agent

    task_model = req.model or _model

    def run_with_lock():
        global _current_task, _current_loop
        with state.busy:
            loop = asyncio.new_event_loop()
            state.active_loop = loop
            _current_loop = loop
            _current_task = loop.create_task(run_agent(req.instructions, task_model, _publisher))
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


@app.post("/cancel")
async def cancel_task():
    if not state.busy.locked():
        raise HTTPException(status_code=409, detail="No task is currently running")

    if _current_task and _current_loop and not _current_task.done():
        _current_loop.call_soon_threadsafe(_current_task.cancel)
        return {"status": "cancelling"}

    raise HTTPException(status_code=409, detail="No cancellable task found")
