# S3 session store — local end-to-end results

Run `044325` at 2026-10-24 04:45 against the kind cluster `komputer-s3-test` (MinIO as S3, Bedrock model `eu.anthropic.claude-haiku-4-5-20251001-v1:0`).

**10/10 scenarios passed** (0–8 in one run; 9, which flips `s3.enabled` on the Helm release, run on its own with `ONLY=9`).

| # | Scenario | Result | Time |
|---|---|---|---|
| 0 | API reports the S3 session store as on | PASS | 0s |
| 1 | PVC agent saves its session at the end of a task | PASS | 9s |
| 2 | Archive: pod + PVC removed, record kept; a new task restores the conversation | PASS | 22s |
| 3 | Ephemeral agent can sleep, and wakes with its conversation | PASS | 15s |
| 4 | Chat history comes back from S3 when Redis has none and there's no pod | PASS | 0s |
| 5 | Deleting an agent saves first; the session outlives the agent | PASS | 0s |
| 6 | A failed save stops the archive (task → Error), force delete still works | PASS | 36s |
| 7 | Schedules can use Archive: each run wakes, restores and re-archives the agent | PASS | 28s |
| 8 | Without S3 on a schedule template the admission policy is installed | PASS | 0s |
| 9 | An agent started before S3 was on gets saved by the operator from outside (opt-in: ONLY=9) | PASS | 96s |

## What the runs found

**Bug, fixed (`240ce29`).** An ephemeral agent that slept got a PVC on its next reconcile: the sleep paths
always wrote `status.pvcName`, which the operator then read as "this agent already has a persistent
workspace". Harmless before S3 (ephemeral agents could never sleep). Reproduced with two reconcile-level
tests, fixed, and confirmed here (scenario 3: no PVC, wake restores from S3).

**Behaviour to know about (not changed).** Sending a task to a sleeping or archived agent without a
`lifecycle` resets its lifecycle to the default (pod stays up). This is the existing wake semantics for
Sleep; it now applies to Archive too. The UI always resends the agent's lifecycle, so it's unaffected;
API/SDK callers and manager agents that omit it will "un-archive" the agent.

**Test-kit issues, fixed.** MinIO first ran on an `emptyDir`, so scaling it down in scenario 6 wiped the
bucket — it now has a PVC. The kit set `operator.leaderElect: false`; during a rolling update two operators
then ran at once and the old one (no S3 credentials yet) overwrote a successful `SessionSaved` with a
failure. The chart default (`true`) is now used, and scenario 9 passes with it.

**Unexplained, not reproduced.** In one run a sleeping ephemeral agent (`eph-042934`) was gone ~13 minutes
after creation, with no delete or finalizer log line. Repeating the same state and cleanup didn't reproduce it.

**Not covered here.** The admission policy's deny path (covered by the operator envtest), real AWS S3 /
IAM roles, squads, and very large sessions.

## 0. API reports the S3 session store as on — PASS

- `GET /config` → 200 `{"s3": {"enabled": true, "bucket": "komputer-sessions"}}`

## 1. PVC agent saves its session at the end of a task — PASS

- task: Complete — 'Task completed'
- sessionStore: `{"lastSavedAt": "2026-10-24T02:43:34Z", "ok": true}`; object `komputer/default/pvc-044325/4d241c79-90cc-4ff6-a198-ba82a23bb33e/session.tar.gz` present
- finalizers: `['komputer.ai/session-store']`

## 2. Archive: pod + PVC removed, record kept; a new task restores the conversation — PASS

- archived; cost kept `0.0059`; pod and PVC gone; object present
- task to an archived agent accepted (woke it)
- second task recalled `ORCA-3` from the restored session (`restored_session: true`); archived again

## 3. Ephemeral agent can sleep, and wakes with its conversation — PASS

- ephemeral agent slept (no PVC, pod deleted)
- woke into a fresh emptyDir, restored from S3, recalled `FALCON-9`, slept again

## 4. Chat history comes back from S3 when Redis has none and there's no pod — PASS

- deleted `komputer-history:arch-044325` in Redis; `GET /events` rebuilt the chat from the S3 object (2117 bytes; had 5140)

## 5. Deleting an agent saves first; the session outlives the agent — PASS

- agent deleted; `komputer/default/pvc-044325/4d241c79-90cc-4ff6-a198-ba82a23bb33e/session.tar.gz` still in the bucket

## 6. A failed save stops the archive (task → Error), force delete still works — PASS

- task: Error — 'Couldn\'t save the session to S3: Could not connect to the endpoint URL: "http://minio.minio.svc.cluster.local:9000/komputer-sessions/komputer/default/fail-044325/36ab92f5-a14a-4139-99f4-a6177c74f015/session.tar.gz"'
- not archived; PVC kept
- `DELETE ?force=true` removed it

## 7. Schedules can use Archive: each run wakes, restores and re-archives the agent — PASS

- run 1: agent archived after the run
- run 2: agent archived after the run

## 8. Without S3 on a schedule template the admission policy is installed — PASS

- `validatingadmissionpolicy.admissionregistration.k8s.io/komputer-schedule-archive-requires-s3` present (deny path covered by the operator envtest)

## 9. An agent started before S3 was on gets saved by the operator from outside (opt-in: ONLY=9) — PASS

- agent ran a task with S3 off; its pod has no S3 env
- turned S3 on
- operator copied the session out on its own after 0s
- `SessionSaved=True`, sessionCopiedOutAt `2026-10-24T02:49:13Z`; object `komputer/default/old-044758/105cb69c-d4b2-43bc-8e4c-1a60f1452cef/session.tar.gz` present
- archived, woke into a fresh PVC from S3, still recalled `HERON-5`
