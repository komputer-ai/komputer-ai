from fastapi import FastAPI, BackgroundTasks
from pydantic import BaseModel

app = FastAPI()

# These get set by main.py before the server starts
_publisher = None
_model = None


def configure(publisher, model: str):
    global _publisher, _model
    _publisher = publisher
    _model = model


class TaskRequest(BaseModel):
    instructions: str


@app.post("/task")
async def create_task(req: TaskRequest, background_tasks: BackgroundTasks):
    from agent import run_agent_sync

    background_tasks.add_task(run_agent_sync, req.instructions, _model, _publisher)
    return {"status": "accepted", "instructions": req.instructions[:100]}
