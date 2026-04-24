"""Agent-side metrics: collected in-process via prometheus_client and optionally
pushed to a Prometheus remote-write endpoint.

Three metrics live here because they describe activity that doesn't appear
in the Redis event stream the API worker consumes:
- steering: when a follow-up message reroutes a running task
- mcp_connector_status: per-connector health at MCP server startup
- subagent_wait_seconds: time spent blocked in wait_for_agents.py

A background flush coroutine pushes these to a Prometheus-compatible
remote-write endpoint every 15s. Disabled when KOMPUTER_METRICS_REMOTE_WRITE_URL
is unset.
"""
import asyncio
import logging
import os
import struct
import time as time_mod
from typing import Optional

import httpx
import snappy
from prometheus_client import Counter, Gauge, Histogram, REGISTRY, generate_latest

logger = logging.getLogger("komputer.agent.metrics")

# Module-level metric handles. Initialized in init().
_steering_total: Optional[Counter] = None
_mcp_status: Optional[Gauge] = None
_subagent_wait: Optional[Histogram] = None
_agent_name_label: str = ""
_flush_task: Optional[asyncio.Task] = None


def _resolve_agent_name() -> str:
    """Returns the agent name when KOMPUTER_METRICS_PER_AGENT=true, "" otherwise.

    Always present as a label so dashboards stay schema-stable across the flag.
    """
    if os.environ.get("KOMPUTER_METRICS_PER_AGENT", "false").lower() == "true":
        return os.environ.get("KOMPUTER_AGENT_NAME", "")
    return ""


def init():
    """Initialize the metric handles. Idempotent — safe to call multiple times.

    Note: prometheus_client's REGISTRY is a process-global singleton, so re-initializing
    requires unregistering existing collectors first. For test isolation we always do that.
    """
    global _steering_total, _mcp_status, _subagent_wait, _agent_name_label
    _agent_name_label = _resolve_agent_name()

    # Unregister any existing collectors with our names (test isolation).
    for name in (
        "komputer_agent_steering_total",
        "komputer_agent_mcp_connector_status",
        "komputer_agent_subagent_wait_seconds",
    ):
        try:
            collector = REGISTRY._names_to_collectors.get(name)
            if collector:
                REGISTRY.unregister(collector)
        except Exception:
            pass

    _steering_total = Counter(
        "komputer_agent_steering_total",
        "Number of times the running agent received a steering follow-up message.",
        labelnames=["agent_name"],
    )
    _mcp_status = Gauge(
        "komputer_agent_mcp_connector_status",
        "MCP connector health: 1 if healthy, 0 if unhealthy.",
        labelnames=["agent_name", "connector", "status"],
    )
    _subagent_wait = Histogram(
        "komputer_agent_subagent_wait_seconds",
        "Wall-clock time spent in wait_for_agents.py blocked on sub-agents.",
        labelnames=["agent_name"],
        buckets=(1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600),
    )


def record_steering():
    if _steering_total is not None:
        _steering_total.labels(agent_name=_agent_name_label).inc()


def set_mcp_status(connector: str, healthy: bool):
    if _mcp_status is not None:
        # Single label "status=healthy" with value 1 (healthy) / 0 (unhealthy)
        # so PromQL queries like `sum by(connector)` are clean.
        _mcp_status.labels(
            agent_name=_agent_name_label, connector=connector, status="healthy"
        ).set(1.0 if healthy else 0.0)


def observe_subagent_wait(seconds: float):
    if _subagent_wait is not None:
        _subagent_wait.labels(agent_name=_agent_name_label).observe(seconds)


def expose_text() -> bytes:
    """Return the current registry as Prometheus text format. Used by the
    /metrics FastAPI route and for the remote-write payload."""
    return generate_latest(REGISTRY)


# --- Remote-write ---


async def _flush_once():
    """Push the current metrics to the remote-write endpoint. Best-effort —
    log failures, never raise."""
    url = os.environ.get("KOMPUTER_METRICS_REMOTE_WRITE_URL", "").strip()
    if not url:
        return
    token = os.environ.get("KOMPUTER_METRICS_REMOTE_WRITE_TOKEN", "").strip()

    try:
        payload = _build_write_request_protobuf()
    except Exception as e:
        logger.warning("metrics: failed to build remote-write payload: %s", e)
        return

    if not payload:
        return  # nothing to push

    compressed = snappy.compress(payload)
    headers = {
        "Content-Type": "application/x-protobuf",
        "Content-Encoding": "snappy",
        "X-Prometheus-Remote-Write-Version": "0.1.0",
    }
    if token:
        headers["Authorization"] = f"Bearer {token}"

    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            resp = await client.post(url, content=compressed, headers=headers)
            if resp.status_code >= 300:
                logger.warning(
                    "metrics: remote-write returned %d: %s",
                    resp.status_code, resp.text[:200],
                )
    except Exception as e:
        logger.warning("metrics: remote-write POST failed: %s", e)


def _build_write_request_protobuf() -> bytes:
    """Build a minimal Prometheus WriteRequest protobuf from the REGISTRY state.

    Wire format (https://prometheus.io/docs/concepts/remote_write_spec/):
      message WriteRequest { repeated TimeSeries timeseries = 1; }
      message TimeSeries { repeated Label labels = 1; repeated Sample samples = 2; }
      message Label { string name = 1; string value = 2; }
      message Sample { double value = 1; int64 timestamp = 2; }

    We hand-roll because the spec is small and stable, avoiding the full
    prometheus protobuf dependency (which bloats the agent image).
    """
    timestamp_ms = int(time_mod.time() * 1000)
    parts = []
    for metric_family in REGISTRY.collect():
        for sample in metric_family.samples:
            # Skip prometheus_client's `_created` timestamp samples — they
            # pollute the remote series set without adding signal.
            if sample.name.endswith("_created"):
                continue
            ts_bytes = _encode_timeseries(sample.name, sample.labels, sample.value, timestamp_ms)
            # field 1 (timeseries), wire type 2 (length-delimited)
            parts.append(_tag_length_value(1, 2, ts_bytes))
    return b"".join(parts)


def _encode_timeseries(name: str, labels: dict, value: float, timestamp_ms: int) -> bytes:
    parts = []
    # The metric name is encoded as the special label __name__.
    all_labels = {"__name__": name, **labels}
    for label_name, label_value in all_labels.items():
        label_bytes = _encode_label(label_name, label_value)
        parts.append(_tag_length_value(1, 2, label_bytes))  # field 1 (labels)
    sample_bytes = _encode_sample(value, timestamp_ms)
    parts.append(_tag_length_value(2, 2, sample_bytes))  # field 2 (samples)
    return b"".join(parts)


def _encode_label(name: str, value: str) -> bytes:
    name_bytes = name.encode("utf-8")
    value_bytes = value.encode("utf-8")
    return _tag_length_value(1, 2, name_bytes) + _tag_length_value(2, 2, value_bytes)


def _encode_sample(value: float, timestamp_ms: int) -> bytes:
    # Sample.value: field 1, wire type 1 (64-bit fixed = double)
    value_part = bytes([(1 << 3) | 1]) + struct.pack("<d", value)
    # Sample.timestamp: field 2, wire type 0 (varint = int64)
    ts_part = bytes([(2 << 3) | 0]) + _encode_varint(timestamp_ms)
    return value_part + ts_part


def _tag_length_value(field_num: int, wire_type: int, data: bytes) -> bytes:
    tag = (field_num << 3) | wire_type
    return _encode_varint(tag) + _encode_varint(len(data)) + data


def _encode_varint(value: int) -> bytes:
    parts = bytearray()
    while value > 0x7F:
        parts.append((value & 0x7F) | 0x80)
        value >>= 7
    parts.append(value & 0x7F)
    return bytes(parts)


# --- Background flush task ---


async def flush_loop(interval: float = 15.0):
    """Background coroutine — flushes metrics every `interval` seconds."""
    while True:
        try:
            await _flush_once()
        except Exception as e:
            logger.warning("metrics: flush loop iteration failed: %s", e)
        await asyncio.sleep(interval)


def start_flush_task(loop: Optional[asyncio.AbstractEventLoop] = None):
    """Start the background flush task. Call once on agent startup.

    No-op when KOMPUTER_METRICS_REMOTE_WRITE_URL is unset (most local dev).
    """
    global _flush_task
    if _flush_task is not None:
        return
    if not os.environ.get("KOMPUTER_METRICS_REMOTE_WRITE_URL", "").strip():
        return  # No remote-write configured, no flush task.
    loop = loop or asyncio.get_event_loop()
    try:
        interval = float(os.environ.get("KOMPUTER_METRICS_REMOTE_WRITE_INTERVAL", "15"))
    except ValueError:
        interval = 15.0
    _flush_task = loop.create_task(flush_loop(interval))
