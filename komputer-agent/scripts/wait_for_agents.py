#!/usr/bin/env python3
"""Block until all specified agents complete, reading directly from Redis streams.

Usage:
    python /app/scripts/wait_for_agents.py agent1 agent2 agent3

Reads Redis config from /etc/komputer/config.json.
Blocks until every agent has a terminal event (task_completed, error, task_cancelled).
Prints progress to stderr and final JSON summary to stdout.
"""

import json
import sys
import time


def load_redis_config():
    try:
        with open("/etc/komputer/config.json") as f:
            return json.load(f).get("redis", {})
    except (FileNotFoundError, json.JSONDecodeError):
        return {}


def field(fields: dict, key: str) -> str:
    val = fields.get(key.encode(), fields.get(key, b""))
    return val.decode() if isinstance(val, bytes) else str(val)


def main():
    import redis as redis_lib

    names = sys.argv[1:]
    if not names:
        print(json.dumps({"error": "No agent names provided"}))
        sys.exit(1)

    config = load_redis_config()
    stream_prefix = config.get("stream_prefix", "komputer-events")
    password = config.get("password") or None

    r = redis_lib.Redis(
        host=config.get("address", "redis:6379").split(":")[0],
        port=int(config.get("address", "redis:6379").split(":")[1]),
        password=password,
        db=config.get("db", 0),
    )

    terminal_types = {"task_completed", "error", "task_cancelled"}

    # Build tracking state.
    pending = {}
    for name in names:
        stream_key = f"{stream_prefix}:{name}"
        pending[name] = {"stream_key": stream_key, "last_id": "0-0", "last_text": ""}

    results = {}
    timeout = 600  # 10 minute hard timeout
    deadline = time.time() + timeout
    total = len(names)

    print(f"Waiting for {total} agent(s): {', '.join(names)}", file=sys.stderr, flush=True)

    while pending and time.time() < deadline:
        streams = {info["stream_key"]: info["last_id"] for info in pending.values()}

        try:
            resp = r.xread(streams, block=5000, count=100)
        except redis_lib.RedisError:
            time.sleep(1)
            continue

        if not resp:
            continue

        for stream_key_bytes, entries in resp:
            stream_key = stream_key_bytes.decode() if isinstance(stream_key_bytes, bytes) else stream_key_bytes

            matched_name = None
            for name, info in pending.items():
                if info["stream_key"] == stream_key:
                    matched_name = name
                    break
            if not matched_name:
                continue

            for entry_id, fields in entries:
                entry_id_str = entry_id.decode() if isinstance(entry_id, bytes) else entry_id
                pending[matched_name]["last_id"] = entry_id_str
                etype = field(fields, "type")

                # Track the last text event as the agent's output.
                if etype == "text":
                    payload_str = field(fields, "payload")
                    try:
                        payload = json.loads(payload_str) if payload_str else {}
                    except json.JSONDecodeError:
                        payload = {}
                    pending[matched_name]["last_text"] = payload.get("content", "")

                if etype in terminal_types:
                    results[matched_name] = {
                        "status": etype,
                        "result": pending[matched_name].get("last_text", ""),
                    }
                    del pending[matched_name]
                    done = total - len(pending)
                    print(f"[{done}/{total}] {matched_name} -> {etype}", file=sys.stderr, flush=True)
                    break

    for name in pending:
        results[name] = {"status": "timeout"}

    summary = {
        "all_complete": len(pending) == 0,
        "completed": len(results),
        "results": results,
    }
    print(json.dumps(summary), flush=True)


if __name__ == "__main__":
    main()
