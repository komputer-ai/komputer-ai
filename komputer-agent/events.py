import json
import logging
import os
import queue
import threading
import time
import redis
from redis.backoff import ExponentialBackoff
from redis.retry import Retry

logger = logging.getLogger(__name__)


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

        # Background publisher: queue + drain thread.
        self._queue: queue.Queue = queue.Queue()
        self._stopped = threading.Event()
        self._worker = threading.Thread(target=self._drain_loop, name="event-publisher", daemon=True)
        self._worker.start()

    def _drain_loop(self):
        """Read events from the queue and publish to Redis.
        The connection pool handles retries and reconnection."""
        while not self._stopped.is_set():
            try:
                stream_key, event = self._queue.get(timeout=0.5)
            except queue.Empty:
                continue
            try:
                self.client.xadd(stream_key, event, maxlen=200, approximate=True)
            except redis.RedisError as e:
                logger.exception("failed to publish event", extra={"error": str(e)})
            self._queue.task_done()

    def ping(self) -> bool:
        """Check if Redis is reachable."""
        try:
            return self.client.ping()
        except redis.RedisError:
            return False

    def publish(self, event_type: str, payload: dict):
        stream_key = f"{self.stream_prefix}:{self.agent_name}"

        event = {
            "agentName": self.agent_name,
            "namespace": self.namespace,
            "type": event_type,
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "payload": json.dumps(payload),
        }

        # Log immediately for kubectl logs visibility.
        logger.debug("publishing event", extra={"event_type": event_type, "agent_name": self.agent_name, "payload": payload})

        # For task_started: trim old messages and publish atomically so the
        # stream only contains events from the current task.
        # Uses XTRIM (not DELETE) to preserve the stream key and its consumer groups.
        # SKIP the trim for steer continuations — they are part of an ongoing task,
        # not a new one, so their preceding events must be kept.
        if event_type == "task_started":
            is_steer = payload.get("steer", False)
            try:
                pipe = self.client.pipeline(transaction=True)
                if not is_steer:
                    pipe.xtrim(stream_key, maxlen=0)
                pipe.xadd(stream_key, event, maxlen=200, approximate=True)
                pipe.execute()
            except redis.RedisError as e:
                logger.exception("failed to publish task_started", extra={"error": str(e)})
            return

        # All other events: enqueue for background publisher — returns immediately.
        self._queue.put((stream_key, event))

    def flush(self):
        """Block until every queued event has been sent."""
        self._queue.join()

    def shutdown(self):
        """Flush remaining events and stop the background thread."""
        self.flush()
        self._stopped.set()
        self._worker.join(timeout=5)
