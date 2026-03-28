import json
import os
import time
import redis


class EventPublisher:
    def __init__(self, redis_config: dict, agent_name: str):
        self.agent_name = agent_name
        self.namespace = os.getenv("KOMPUTER_NAMESPACE", "default")
        self.stream_prefix = redis_config.get("stream_prefix", "komputer-events")
        password = redis_config.get("password") or None
        self.client = redis.Redis(
            host=redis_config["address"].split(":")[0],
            port=int(redis_config["address"].split(":")[1]),
            password=password,
            db=redis_config.get("db", 0),
        )

    def publish(self, event_type: str, payload: dict):
        stream_key = f"{self.stream_prefix}:{self.agent_name}"

        # Clear the stream at the start of each task so previous events
        # don't confuse the worker or CLI catch-up on wake-up.
        if event_type == "task_started":
            try:
                self.client.delete(stream_key)
            except redis.RedisError:
                pass

        event = {
            "agentName": self.agent_name,
            "namespace": self.namespace,
            "type": event_type,
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "payload": json.dumps(payload),
        }
        # Log every event as JSON for pod log visibility (kubectl logs).
        log_entry = {**event, "payload": payload}
        print(json.dumps(log_entry), flush=True)

        self.client.xadd(stream_key, event, maxlen=200, approximate=True)
