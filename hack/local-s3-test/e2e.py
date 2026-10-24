#!/usr/bin/env python3
"""End-to-end scenarios for the S3 session store against the local kind cluster.

Drives the real API (http://localhost:28080), checks objects directly in MinIO
(http://localhost:29000), and writes a markdown report. Needs boto3.

    KUBECONFIG=~/.kube/komputer-s3-test.yaml python3 hack/local-s3-test/e2e.py
    MODEL=eu.anthropic.claude-sonnet-5 ... to use another Bedrock model
"""
import json
import os
import subprocess
import sys
import time
import traceback
import urllib.error
import urllib.request
from datetime import datetime

import boto3

API = os.environ.get("API", "http://localhost:28080/api/v1")
MODEL = os.environ.get("MODEL", "eu.anthropic.claude-haiku-4-5-20251001-v1:0")
NS = "default"
RUN = os.environ.get("RUN") or datetime.now().strftime("%H%M%S")
ONLY = {x.strip() for x in os.environ.get("ONLY", "").split(",") if x.strip()}
REPORT = os.environ.get("REPORT", os.path.join(os.path.dirname(__file__), "RESULTS.md"))
KUBECONFIG = os.environ.get("KUBECONFIG", os.path.expanduser("~/.kube/komputer-s3-test.yaml"))

s3 = boto3.client("s3", endpoint_url="http://localhost:29000", region_name="us-east-1",
                  aws_access_key_id="minioadmin", aws_secret_access_key="minioadmin-local")
BUCKET = "komputer-sessions"

results = []


def api(method, path, body=None, ok=(200, 201)):
    req = urllib.request.Request(API + path, method=method,
                                 data=json.dumps(body).encode() if body is not None else None,
                                 headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            raw = r.read()
            return r.status, (json.loads(raw) if raw else None)
    except urllib.error.HTTPError as e:
        raw = e.read()
        try:
            data = json.loads(raw)
        except Exception:
            data = raw.decode(errors="replace")
        if e.code in ok:
            return e.code, data
        return e.code, data


def kubectl(*args, check=True):
    out = subprocess.run(["kubectl", "--kubeconfig", KUBECONFIG, *args], capture_output=True, text=True)
    if check and out.returncode != 0:
        raise RuntimeError(f"kubectl {' '.join(args)}: {out.stderr.strip()}")
    return out.stdout.strip()


def agent(name):
    code, a = api("GET", f"/agents/{name}?namespace={NS}")
    return a if code == 200 else None


def cr(name):
    out = kubectl("get", "komputeragent", name, "-n", NS, "-o", "json", check=False)
    return json.loads(out) if out else None


def wait(desc, pred, timeout=420, every=3):
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        try:
            last = pred()
            if last:
                return last
        except Exception as e:  # noqa: BLE001
            last = e
        time.sleep(every)
    raise TimeoutError(f"timed out waiting for {desc} (last: {last})")


def task_done(name):
    def pred():
        a = agent(name)
        return a if a and a.get("taskStatus") in ("Complete", "Error") and a.get("status") not in ("Pending",) else None
    return pred


def objects(prefix):
    resp = s3.list_objects_v2(Bucket=BUCKET, Prefix=prefix)
    return [o["Key"] for o in resp.get("Contents", [])]


def session_key(name):
    c = cr(name)
    return f"komputer/{NS}/{name}/{c['metadata']['uid']}/latest.tar.gz" if c else None


def exists(kind, name, ns=NS):
    return bool(kubectl("get", kind, name, "-n", ns, "--ignore-not-found", "-o", "name", check=False))


def events_text(name):
    code, data = api("GET", f"/agents/{name}/events?namespace={NS}&limit=200")
    return json.dumps(data) if code == 200 else ""


def create(name, instructions, **extra):
    body = {"name": name, "instructions": instructions, "model": MODEL, "role": "worker", "namespace": NS}
    body.update(extra)
    code, data = api("POST", "/agents", body)
    if code not in (200, 201):
        raise RuntimeError(f"create {name}: {code} {data}")
    return data


def send(name, instructions, lifecycle=""):
    # Like the UI chat: a message to an existing agent carries its lifecycle
    # (a wake with none resets it to the default, as for Sleep agents).
    code, data = api("POST", "/agents", {"name": name, "instructions": instructions, "namespace": NS, "lifecycle": lifecycle})
    if code not in (200, 201):
        raise RuntimeError(f"task to {name}: {code} {data}")
    return data


def scenario(title):
    def deco(fn):
        def run():
            start = time.time()
            notes = []
            try:
                fn(notes)
                results.append((title, "PASS", time.time() - start, notes))
            except Exception as e:  # noqa: BLE001
                notes.append(f"**Failed:** `{e}`")
                notes.append("```\n" + traceback.format_exc()[-1500:] + "\n```")
                results.append((title, "FAIL", time.time() - start, notes))
            print(f"[{results[-1][1]}] {title} ({results[-1][2]:.0f}s)", flush=True)
        run.__name__ = fn.__name__
        return run
    return deco


def check(cond, msg):
    if not cond:
        raise AssertionError(msg)


# ---------------------------------------------------------------------------

@scenario("0. API reports the S3 session store as on")
def s0(notes):
    code, cfg = api("GET", "/config")
    notes.append(f"`GET /config` → {code} `{json.dumps(cfg)}`")
    check(code == 200 and cfg["s3"]["enabled"] and cfg["s3"]["bucket"] == BUCKET, "s3 should be enabled")
    check("credentials" not in json.dumps(cfg).lower(), "config must not leak credentials")


@scenario("1. PVC agent saves its session at the end of a task")
def s1(notes):
    name = f"pvc-{RUN}"
    create(name, "Remember the codeword PELICAN-7. Reply with just OK.")
    a = wait("task end", task_done(name))
    notes.append(f"task: {a['taskStatus']} — {a.get('lastTaskMessage')!r}")
    check(a["taskStatus"] == "Complete", f"task should complete, got {a['taskStatus']}: {a.get('lastTaskMessage')}")
    check(a.get("sessionStore", {}).get("ok"), f"sessionStore should be ok: {a.get('sessionStore')}")
    key = session_key(name)
    check(key in objects(f"komputer/{NS}/{name}/"), f"object {key} should exist")
    notes.append(f"sessionStore: `{json.dumps(a['sessionStore'])}`; object `{key}` present")
    fin = cr(name)["metadata"].get("finalizers", [])
    check("komputer.ai/session-store" in fin, f"finalizer missing: {fin}")
    notes.append(f"finalizers: `{fin}`")


@scenario("2. Archive: pod + PVC removed, record kept; a new task restores the conversation")
def s2(notes):
    name = f"arch-{RUN}"
    create(name, "Remember the codeword ORCA-3. Reply with just OK.", lifecycle="Archive")
    wait("archived", lambda: (agent(name) or {}).get("status") == "Archived", timeout=480)
    check(not exists("pod", f"{name}-pod"), "pod should be gone")
    check(not exists("pvc", f"{name}-pvc"), "PVC should be gone")
    key = session_key(name)
    check(key in objects(f"komputer/{NS}/{name}/"), "session object should exist")
    a = agent(name)
    notes.append(f"archived; cost kept `{a.get('totalCostUSD')}`; pod and PVC gone; object present")

    send(name, "What codeword did I ask you to remember? Reply with just the codeword.", "Archive")
    wait("woken", lambda: (agent(name) or {}).get("status") != "Archived", timeout=60)
    notes.append("task to an archived agent accepted (woke it)")
    wait("re-archived", lambda: (agent(name) or {}).get("status") == "Archived" and (agent(name) or {}).get("taskStatus") == "Complete", timeout=480)
    text = events_text(name)
    check("ORCA-3" in text.split('"task_started"')[-1] if '"task_started"' in text else "ORCA-3" in text, "agent should recall ORCA-3 after restore")
    check('"restored_session": true' in text or '"restored_session":true' in text, "task_started should report restored_session=true")
    notes.append("second task recalled `ORCA-3` from the restored session (`restored_session: true`); archived again")


@scenario("3. Ephemeral agent can sleep, and wakes with its conversation")
def s3_ephemeral(notes):
    name = f"eph-{RUN}"
    create(name, "Remember the codeword FALCON-9. Reply with just OK.", lifecycle="Sleep", storage={"ephemeral": True})
    wait("sleeping", lambda: (agent(name) or {}).get("status") == "Sleeping", timeout=480)
    # Sleeping is recorded when the pod is deleted; it terminates within its grace period.
    wait("pod gone", lambda: not exists("pod", f"{name}-pod"), timeout=120)
    check(not exists("pvc", f"{name}-pvc"), "an ephemeral agent has no PVC")
    notes.append("ephemeral agent slept (no PVC, pod deleted)")
    send(name, "What codeword did I ask you to remember? Reply with just the codeword.", "Sleep")
    wait("asleep again", lambda: (agent(name) or {}).get("status") == "Sleeping" and "FALCON" in events_text(name).split('"restored_session"')[-1], timeout=480)
    check('"restored_session": true' in events_text(name) or '"restored_session":true' in events_text(name), "wake should restore from S3")
    notes.append("woke into a fresh emptyDir, restored from S3, recalled `FALCON-9`, slept again")


@scenario("4. Chat history comes back from S3 when Redis has none and there's no pod")
def s4(notes):
    name = f"arch-{RUN}"
    before = events_text(name)
    redis_pod = kubectl("get", "pod", "-n", "komputer-system", "-l", "app.kubernetes.io/component=redis", "-o", "jsonpath={.items[0].metadata.name}")
    kubectl("exec", "-n", "komputer-system", redis_pod, "--", "redis-cli", "DEL", f"komputer-history:{name}")
    check((agent(name) or {}).get("status") == "Archived", "agent should be archived (no pod)")
    after = events_text(name)
    check("ORCA-3" in after, f"history should be rebuilt from S3 (got {after[:300]})")
    notes.append(f"deleted `komputer-history:{name}` in Redis; `GET /events` rebuilt the chat from the S3 object ({len(after)} bytes; had {len(before)})")


@scenario("5. Deleting an agent saves first; the session outlives the agent")
def s5(notes):
    name = f"pvc-{RUN}"
    key = session_key(name)
    code, _ = api("DELETE", f"/agents/{name}?namespace={NS}")
    check(code == 200, f"delete returned {code}")
    wait("deleted", lambda: cr(name) is None, timeout=180)
    check(key in objects(f"komputer/{NS}/{name}/"), "session object must remain after delete")
    notes.append(f"agent deleted; `{key}` still in the bucket")


@scenario("6. A failed save stops the archive (task → Error), force delete still works")
def s6(notes):
    name = f"fail-{RUN}"
    kubectl("scale", "deploy/minio", "-n", "minio", "--replicas=0")
    try:
        wait("minio down", lambda: kubectl("get", "pods", "-n", "minio", "-l", "app=minio", "-o", "name", check=False) == "", timeout=120)
        create(name, "Reply with just OK.", lifecycle="Archive")
        a = wait("task end", task_done(name), timeout=480)
        notes.append(f"task: {a['taskStatus']} — {a.get('lastTaskMessage')!r}")
        check(a["taskStatus"] == "Error", "task should be marked Error when the save fails")
        check("Couldn't save the session to S3" in (a.get("lastTaskMessage") or ""), "message should explain the failed save")
        time.sleep(10)
        check((agent(name) or {}).get("status") != "Archived", "must not archive without a saved session")
        check(exists("pvc", f"{name}-pvc"), "PVC must be kept")
        notes.append("not archived; PVC kept")
    finally:
        kubectl("scale", "deploy/minio", "-n", "minio", "--replicas=1")
        kubectl("rollout", "status", "deploy/minio", "-n", "minio", "--timeout=180s")
    code, _ = api("DELETE", f"/agents/{name}?namespace={NS}&force=true")
    check(code == 200, f"force delete returned {code}")
    wait("force-deleted", lambda: cr(name) is None, timeout=120)
    notes.append("`DELETE ?force=true` removed it")


@scenario("7. Schedules can use Archive: each run wakes, restores and re-archives the agent")
def s7(notes):
    name = f"sched-{RUN}"
    body = {"name": name, "namespace": NS, "schedule": "0 0 1 1 *", "instructions": "Reply with just the word TICK.",
            "agent": {"lifecycle": "Archive", "model": MODEL, "role": "worker"}}
    code, data = api("POST", "/schedules", body)
    check(code in (200, 201), f"create schedule: {code} {data}")
    agent_name = f"{name}-agent"
    for i in (1, 2):
        code, data = api("POST", f"/schedules/{name}/trigger?namespace={NS}")
        check(code in (200, 201, 202), f"trigger {i}: {code} {data}")
        wait(f"run {i} archived", lambda: (agent(agent_name) or {}).get("status") == "Archived" and (agent(agent_name) or {}).get("taskStatus") == "Complete", timeout=480)
        notes.append(f"run {i}: agent archived after the run")
        time.sleep(5)
    api("DELETE", f"/schedules/{name}?namespace={NS}")


@scenario("8. Without S3 on a schedule template the admission policy is installed")
def s8(notes):
    out = kubectl("get", "validatingadmissionpolicy", "komputer-schedule-archive-requires-s3", "-o", "name", check=False)
    check(out, "the ValidatingAdmissionPolicy should be installed (Kubernetes >= 1.30)")
    notes.append(f"`{out}` present (deny path covered by the operator envtest)")


HERE = os.path.dirname(os.path.abspath(__file__))
CHART = os.path.join(HERE, "..", "..", "helm", "komputer-ai")


def helm_s3(enabled):
    out = subprocess.run(["helm", "--kubeconfig", KUBECONFIG, "upgrade", "komputer-ai", CHART, "-n", "komputer-system",
                          "-f", os.path.join(HERE, "values.yaml"), "--set", f"s3.enabled={str(enabled).lower()}",
                          "--wait", "--timeout", "5m"], capture_output=True, text=True)
    if out.returncode != 0:
        raise RuntimeError(f"helm upgrade: {out.stderr.strip()}")
    # Same as up.sh: Bedrock credentials for agents (helm may drop the live patch).
    kubectl("patch", "komputeragentclustertemplate", "default", "--type=json", "-p",
            '[{"op":"add","path":"/spec/podSpec/containers/0/envFrom","value":[{"secretRef":{"name":"komputer-aws-creds"}}]}]')
    wait(f"API sees s3={enabled}", lambda: api("GET", "/config")[1]["s3"]["enabled"] == enabled, timeout=60)


@scenario("9. An agent started before S3 was on gets saved by the operator from outside (opt-in: ONLY=9)")
def s9(notes):
    name = f"old-{RUN}"
    helm_s3(False)
    try:
        create(name, "Remember the codeword HERON-5. Reply with just OK.")
        a = wait("task end", task_done(name))
        check(a["taskStatus"] == "Complete", f"task: {a['taskStatus']} {a.get('lastTaskMessage')}")
        env = kubectl("get", "pod", f"{name}-pod", "-n", NS, "-o", "jsonpath={.spec.containers[0].env[*].name}")
        check("KOMPUTER_S3_BUCKET" not in env, "a pod started with S3 off has no S3 env")
        notes.append("agent ran a task with S3 off; its pod has no S3 env")
    finally:
        helm_s3(True)
    notes.append("turned S3 on")
    key = session_key(name)
    start = time.time()
    try:
        wait("copy-out", lambda: key in objects(f"komputer/{NS}/{name}/"), timeout=150)
        notes.append(f"operator copied the session out on its own after {time.time() - start:.0f}s")
    except TimeoutError:
        notes.append("not copied out within 150s of turning S3 on (no KomputerConfig watch — known deferred minor); nudging the agent")
        kubectl("annotate", "komputeragent", name, "-n", NS, f"e2e/nudge={int(time.time())}", "--overwrite")
        wait("copy-out after nudge", lambda: key in objects(f"komputer/{NS}/{name}/"), timeout=150)
        notes.append(f"copied out {time.time() - start:.0f}s after S3 was turned on (after the nudge)")
    c = cr(name)
    conds = {x["type"]: x for x in c["status"].get("conditions", [])}
    check(conds.get("SessionSaved", {}).get("status") == "True", f"SessionSaved condition: {conds.get('SessionSaved')}")
    check(c["status"].get("sessionCopiedOutAt"), "sessionCopiedOutAt should be set")
    notes.append(f"`SessionSaved=True`, sessionCopiedOutAt `{c['status']['sessionCopiedOutAt']}`; object `{key}` present")
    # The saved copy is good: archive it and ask again.
    code, data = api("PATCH", f"/agents/{name}?namespace={NS}", {"lifecycle": "Archive"})
    check(code == 200, f"patch lifecycle: {code} {data}")
    send(name, "What codeword did I ask you to remember? Reply with just the codeword.", "Archive")
    wait("archived", lambda: (agent(name) or {}).get("status") == "Archived", timeout=480)
    send(name, "What codeword did I ask you to remember? Reply with just the codeword.", "Archive")
    wait("woken", lambda: (agent(name) or {}).get("status") != "Archived", timeout=60)
    wait("re-archived", lambda: (agent(name) or {}).get("status") == "Archived" and (agent(name) or {}).get("taskStatus") == "Complete", timeout=480)
    text = events_text(name)
    check(text.count("HERON-5") >= 3, "the restored agent should still recall HERON-5")
    notes.append("archived, woke into a fresh PVC from S3, still recalled `HERON-5`")


@scenario("10. A corrupt latest save falls back to the previous one")
def s10(notes):
    name = f"prev-{RUN}"
    create(name, "Remember the codeword PUFFIN-2. Reply with just OK.", lifecycle="Sleep", storage={"ephemeral": True})
    wait("asleep", lambda: (agent(name) or {}).get("status") == "Sleeping", timeout=480)
    wait("pod gone", lambda: not exists("pod", f"{name}-pod"), timeout=120)
    latest = session_key(name)
    previous = latest.rsplit("/", 1)[0] + "/previous.tar.gz"
    check(previous not in objects(f"komputer/{NS}/{name}/"), "first save has nothing to keep as previous")
    send(name, "Reply with just OK.", "Sleep")
    wait("woken", lambda: (agent(name) or {}).get("status") != "Sleeping", timeout=60)
    wait("asleep again", lambda: (agent(name) or {}).get("status") == "Sleeping", timeout=480)
    wait("pod gone", lambda: not exists("pod", f"{name}-pod"), timeout=120)
    check(previous in objects(f"komputer/{NS}/{name}/"), "second save keeps the first as previous")
    notes.append("two saves: `latest.tar.gz` and `previous.tar.gz` both present")
    s3.put_object(Bucket=BUCKET, Key=latest, Body=b"not a tarball")
    notes.append("overwrote `latest.tar.gz` with garbage")
    send(name, "What codeword did I ask you to remember? Reply with just the codeword.", "Sleep")
    wait("woken", lambda: (agent(name) or {}).get("status") != "Sleeping", timeout=60)
    wait("asleep again", lambda: (agent(name) or {}).get("status") == "Sleeping", timeout=480)
    text = events_text(name)
    last = text.split('"task_started"')[-1]
    check('"restored_from": "previous"' in last, "the wake should restore from previous")
    check("PUFFIN-2" in last, "the agent should recall PUFFIN-2 from the previous save")
    with __import__("io").BytesIO() as b:
        s3.download_fileobj(BUCKET, latest, b)
        check(b.getvalue()[:2] == b"\x1f\x8b", "the next save should leave a good latest again")
    notes.append("wake reported `restored_from: previous`, recalled `PUFFIN-2`; its save left a good `latest` again")


def main():
    print(f"API={API} MODEL={MODEL} RUN={RUN}", flush=True)
    for i, fn in enumerate((s0, s1, s2, s3_ephemeral, s4, s5, s6, s7, s8, s9, s10)):
        # 9 flips the Helm release's s3.enabled, so it only runs when asked for.
        if (not ONLY and i != 9) or str(i) in ONLY:
            fn()
    passed = sum(1 for r in results if r[1] == "PASS")
    lines = [
        "# S3 session store — local end-to-end results",
        "",
        f"Run `{RUN}` at {datetime.now().strftime('%Y-%m-%d %H:%M')} against the kind cluster `komputer-s3-test` "
        f"(MinIO as S3, Bedrock model `{MODEL}`).",
        "",
        f"**{passed}/{len(results)} scenarios passed.**",
        "",
        "| # | Scenario | Result | Time |",
        "|---|---|---|---|",
    ]
    for title, status, secs, _ in results:
        lines.append(f"| {title.split('.')[0]} | {title.split('. ', 1)[1]} | {status} | {secs:.0f}s |")
    for title, status, secs, notes in results:
        lines += ["", f"## {title} — {status}", ""] + [f"- {n}" if not n.startswith("```") else n for n in notes]
    with open(REPORT, "w") as f:
        f.write("\n".join(lines) + "\n")
    print(f"\nreport: {REPORT}")
    sys.exit(0 if passed == len(results) else 1)


if __name__ == "__main__":
    main()
