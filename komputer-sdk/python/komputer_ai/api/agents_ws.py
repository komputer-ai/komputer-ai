"""WebSocket streaming for agents — hand-written, preserved across regeneration."""

import json
from dataclasses import dataclass, field
from typing import Iterator

import websocket


@dataclass
class AgentEvent:
    """A single event from an agent's WebSocket stream."""

    agent_name: str
    type: str
    timestamp: str = ""
    payload: dict = field(default_factory=dict)


class AgentEventStream:
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

    def __iter__(self) -> Iterator[AgentEvent]:
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
                payload=data.get("payload", {}),
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
