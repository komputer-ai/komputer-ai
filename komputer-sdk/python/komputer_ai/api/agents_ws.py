"""WebSocket streaming for agents — hand-written, preserved across regeneration."""

import json
from dataclasses import dataclass, field
from typing import Iterator, Self

import websocket


class Payload(dict):
    """Dict subclass that supports attribute access (event.payload.content)."""

    def __getattr__(self, key):
        try:
            return self[key]
        except KeyError:
            return None


@dataclass
class AgentEvent:
    """A single event from an agent's WebSocket stream.

    Access payload fields with dot notation: event.payload.content

    Payload fields by event type:
        task_started:   instructions, resuming_session
        thinking:       content, usage
        text:           content, usage
        tool_call:      id, tool, input
        tool_result:    tool, input, output
        task_completed: cost_usd, duration_ms, turns, stop_reason, session_id, usage
        task_cancelled: reason
        error:          error
    """

    agent_name: str
    type: str
    timestamp: str = ""
    payload: Payload = field(default_factory=Payload)


class AgentEventStream(Iterator[AgentEvent]):
    """Iterable stream of agent events over WebSocket.

    Usage:
        stream = AgentEventStream("ws://localhost:8080", "my-agent")
        for event in stream:
            print(event.type, event.payload)
    """

    def __init__(self, ws_url: str, agent_name: str):
        self._agent_name = agent_name
        self._ws = websocket.WebSocket()
        self._ws.connect(f"{ws_url}/api/v1/agents/{agent_name}/ws")

    def __iter__(self) -> Self:
        return self

    def __next__(self) -> AgentEvent:
        try:
            raw = self._ws.recv()
            if not raw:
                raise StopIteration
            data = json.loads(raw)
            return AgentEvent(
                agent_name=data.get("agentName", self._agent_name),
                type=data.get("type", ""),
                timestamp=data.get("timestamp", ""),
                payload=Payload(data.get("payload", {})),
            )
        except Exception:
            self.close()
            raise StopIteration

    def close(self):
        try:
            self._ws.close()
        except Exception:
            pass

    def __enter__(self):
        return self

    def __exit__(self, *args):
        self.close()
