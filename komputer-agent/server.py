import threading

from fastapi import BackgroundTasks, FastAPI, HTTPException
from pydantic import BaseModel

app = FastAPI()

# These get set by main.py before the server starts
_publisher = None
_model = None
_busy = threading.Lock()


def configure(publisher, model: str):
    global _publisher, _model
    _publisher = publisher
    _model = model


class TaskRequest(BaseModel):
    instructions: str


@app.get("/status")
async def get_status():
    busy = _busy.locked()
    return {"busy": busy}


@app.post("/task")
async def create_task(req: TaskRequest, background_tasks: BackgroundTasks):
    if _busy.locked():
        raise HTTPException(status_code=409, detail="Agent is busy with another task")

    from agent import run_agent_sync

    def run_with_lock():
        with _busy:
            run_agent_sync(req.instructions, _model, _publisher)

    background_tasks.add_task(run_with_lock)
    return {"status": "accepted", "instructions": req.instructions[:100]}
