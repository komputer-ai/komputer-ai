import json
import os
import time
import redis
from redis.backoff import ExponentialBackoff
from redis.retry import Retry


class EventPublisher:
    def __init__(self, redis_config: dict, agent_name: str):
        self.agent_name = agent_name
        self.namespace = os.getenv("KOMPUTER_NAMESPACE", "default")
        self.stream_prefix = redis_config.get("stream_prefix", "komputer-events")
        password = redis_config.get("password") or None

        # Connection pool with health checks and retry on connection errors.
        retry = Retry(ExponentialBackoff(cap=5, base=0.1), retries=3)
        self.pool = redis.ConnectionPool(
            host=redis_config["address"].split(":")[0],
            port=int(redis_config["address"].split(":")[1]),
            password=password,
            db=redis_config.get("db", 0),
            health_check_interval=30,
            socket_connect_timeout=5,
            socket_timeout=10,
            retry=retry,
            retry_on_error=[redis.ConnectionError, redis.TimeoutError],
        )
        self.client = redis.Redis(connection_pool=self.pool)

    def ping(self) -> bool:
        """Check if Redis is reachable."""
        try:
            return self.client.ping()
        except redis.RedisError:
            return False

    def publish(self, event_type: str, payload: dict):
        stream_key = f"{self.stream_prefix}:{self.agent_name}"

        # Clear the stream at the start of each task so the worker and
        # CLI only see events from the current task.
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

        # Publish to real-time stream. The API worker appends to history as it consumes.
        self.client.xadd(stream_key, event, maxlen=200, approximate=True)
