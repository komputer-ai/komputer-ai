---
title: S3 session store
description: Keep every agent's conversation in S3 so agents without a workspace volume can come back exactly where they left off.
---

An agent's conversation — Claude Code's session transcript and the pointer it resumes from — lives in `/workspace/.claude`. Without the session store, it's only as durable as the workspace volume: an ephemeral agent loses it on every pod restart, and Archive deletes it with the PVC.

With the **S3 session store** configured, every agent saves its session to S3 at the end of every task, and restores it when it starts without one. Nothing changes for agents that keep their PVC — their own disk already has the session — but agents without one can now sleep, be archived, and come back with their conversation intact.

Any S3-compatible store works: AWS S3, MinIO, Cloudflare R2, GCS (interoperability mode), Ceph.

## What's stored

Two objects per agent:

```
s3://<bucket>/<prefix>/<namespace>/<agent-name>/<agent-uid>/latest.tar.gz
s3://<bucket>/<prefix>/<namespace>/<agent-name>/<agent-uid>/previous.tar.gz
```

Every save first copies `latest` to `previous` (server side), then replaces `latest`. Restores read `latest`; if it can't be read (a corrupt save), they fall back to `previous` and the task-started event reports `restored_from: previous`. A failed download never falls back — that could quietly roll the agent back to an older session.

Each holds `/workspace/.claude` (minus `skills/`, which is regenerated on every start) and `/workspace/.komputer-session`. **Workspace files are not stored** — cloned repos, generated files, installed packages. An agent that needs its files across restarts should keep a PVC (`lifecycle: Sleep` or the default).

The agent's UID is part of the key, so a new agent that reuses a deleted agent's name never resumes the old conversation.

## Turning it on

Set the `s3` block in the Helm values. It populates `KomputerConfig.spec.s3`, and the operator passes it to every agent.

```yaml
s3:
  enabled: true
  bucket: my-komputer-sessions
  region: us-east-1
  prefix: komputer
  credentials:
    secretName: komputer-s3   # or leave empty to use IAM roles (below)
```

The operator and the API use the store too — the operator to save on an agent's behalf (older pods, deletes), the API to rebuild chat history for an agent that has no pod — so all three need access.

### Static keys

Create a Secret with the keys `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` in the release namespace and set `s3.credentials.secretName`:

```bash
kubectl create secret generic komputer-s3 -n komputer-ai \
  --from-literal=AWS_ACCESS_KEY_ID=... --from-literal=AWS_SECRET_ACCESS_KEY=...
```

The chart injects it into the operator and the API, and the operator copies it into every agent namespace (the same way it copies the Anthropic key) and gives agents the keys as `KOMPUTER_S3_ACCESS_KEY_ID`/`KOMPUTER_S3_SECRET_ACCESS_KEY`. A missing Secret never stops pods from starting — saves just fail, and the agent shows it.

### IAM roles (EKS IRSA / Pod Identity, GKE Workload Identity)

Leave `s3.credentials.secretName` empty. Everything uses the AWS SDK's default credential chain, so give each component's ServiceAccount the role:

```yaml
operator:
  serviceAccount:
    annotations:
      eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/komputer-sessions
api:
  serviceAccount:
    annotations:
      eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/komputer-sessions
```

For agents, set `podSpec.serviceAccountName` on the agent template to a ServiceAccount with the role. Squad pods use it too when every member's template names the same ServiceAccount (a pod can only have one; mixed squads run without one and say so in the squad's status).

### Other credentials

The store's credentials use their own env var names, `KOMPUTER_S3_ACCESS_KEY_ID` and `KOMPUTER_S3_SECRET_ACCESS_KEY`, so they never collide with an agent's other AWS use (such as Bedrock's `AWS_REGION` and credentials). To provide credentials some other way, leave `s3.credentials.secretName` empty and either:

- **Set the store's own variables yourself** — `KOMPUTER_S3_ACCESS_KEY_ID`/`KOMPUTER_S3_SECRET_ACCESS_KEY` from your own Secret, in the agent template's `podSpec.containers[0].env` for agents and in Helm `operator.extraEnv` / `api.extraEnv` for the operator and API; or
- **Rely on the AWS default credential chain** — when the `KOMPUTER_S3_*` keys aren't set, the store uses whatever the AWS SDK finds: `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `AWS_ROLE_ARN` + `AWS_WEB_IDENTITY_TOKEN_FILE`, `AWS_PROFILE`, or an IAM role on the ServiceAccount. These are shared with anything else in the pod that uses AWS.

When `secretName` is set, the injected `KOMPUTER_S3_*` variables take precedence over template env vars with the same name. Credentials are read when a pod starts, so after rotating them restart the operator, the API, and the agents.

### MinIO, R2 and other stores

Set the endpoint, and usually path-style addressing:

```yaml
s3:
  enabled: true
  bucket: komputer
  region: us-east-1
  endpoint: http://minio.minio.svc.cluster.local:9000
  forcePathStyle: true
  credentials:
    secretName: komputer-s3
```

For R2, use `endpoint: https://<account-id>.r2.cloudflarestorage.com` and `region: auto`.

### Least-privilege IAM

```json
{
  "Version": "2012-10-17",
  "Statement": [
    { "Effect": "Allow", "Action": ["s3:GetObject", "s3:PutObject"], "Resource": "arn:aws:s3:::my-komputer-sessions/komputer/*" },
    { "Effect": "Allow", "Action": "s3:ListBucket", "Resource": "arn:aws:s3:::my-komputer-sessions",
      "Condition": { "StringLike": { "s3:prefix": "komputer/*" } } }
  ]
}
```

No `s3:DeleteObject` — komputer never deletes from the bucket. The API only needs `s3:GetObject`. Session transcripts contain everything the agent saw, so keep the bucket private and turn on default encryption (SSE-S3 or KMS).

## How it behaves

- **Saving** happens at the end of every task, before the task is reported done — so by the time an agent shows `Complete`, its session is in S3. A pod that's shutting down interrupts its task, and the interrupted task is saved the same way (with the store on, agent pods get a 60s termination grace period unless the template sets one).
- **Restoring** happens only when the agent starts without a local session. An agent with a PVC never restores.
- **A failed save never fails the task.** The agent shows **Session save failed** with the reason. It only blocks what would destroy the last local copy (below).

| Situation | With the S3 session store |
|---|---|
| PVC agent, any task | saved; never restored (its disk wins) |
| PVC agent loses its PVC | restored from S3 |
| Ephemeral agent restarts | restored — the conversation survives, files don't |
| Ephemeral agent sleeps (`lifecycle: Sleep`, `sleepTTL`, sleep now) | allowed once its latest task is saved; the wake restores it |
| Archived agent | works like a sleeping agent with no workspace (below) |
| Schedule with `lifecycle: Archive` | allowed — each run wakes the agent, restores it, and archives it again |
| Agent deleted (any way) | saved first, then deleted; the session stays in the bucket |

### Nothing destroys the last copy before it's saved

Archive (which deletes the PVC), sleeping an ephemeral agent, AutoDelete, and any delete all wait until the latest task's session is saved:

- If the agent's own save fails and its lifecycle is Archive or AutoDelete, the task is marked **Error** with the reason (for example `Couldn't save the session to S3: AccessDenied`), so the transition doesn't happen. Fix the store, then send the agent its next task — or delete it.
- **Deletes wait.** While the store is on, agents carry a `komputer.ai/session-store` finalizer. On delete, the operator cancels a running task (the agent saves as it ends), or saves the session itself. An agent that only has a PVC left gets a short-lived **saver pod** that mounts it read-only; at most `operator.sessionSaver.maxConcurrent` (default 4) saver pods run at once. If saving keeps failing, the delete waits and the agent's status says why.
- **To delete without saving** (for example the bucket is gone): `komputer agents delete <name> --force`, `DELETE /api/v1/agents/<name>?force=true`, or **Delete without saving** in the UI (shown when the last save failed).

### Archived agents

With the store on, an archived agent is a sleeping agent with no workspace and no storage cost. Every control works and takes effect the next time it runs: send it a task or wake it (it gets a fresh workspace and restores its conversation), change any setting, add it to a squad. If its lifecycle is still Archive, it archives again after its next task.

The agents list hides archived agents unless you pick the **Archived** filter.

Without the store, Archive stays a read-only record (see [Agents](./agents.md#archived-agents)).

### Agents that were running before the store was turned on

Pods created before the store was configured (or running an older agent image) can't save themselves. The operator saves them from the outside: whenever such an agent is idle with an unsaved task, it streams the session out of the container with `tar` and uploads it with its own credentials. Once the pod is recreated it saves itself.

### Chat history

The UI reads chat history from Redis, as before. When Redis has no history for an agent and the agent has no pod (sleeping or archived), the API reads the transcript out of the agent's saved session in S3 and refills Redis — so the chat is never empty just because the bundled Redis restarted.

## Turning it off, or changing buckets

- **Off:** agents stop saving and restoring; ephemeral agents go back to ignoring Sleep, archived agents become read-only again, and Archive schedules stop running (the schedule goes to `Error`). An ephemeral agent that's asleep at that moment wakes with a fresh conversation. The finalizer removes itself on the next delete.
- **New bucket or prefix:** every agent starts fresh under the new location. To keep sessions, copy the objects across first (`aws s3 sync s3://old/komputer s3://new/komputer`).

## Retention

Sessions outlive their agents, so you can audit what any agent did. Expire them with a bucket lifecycle rule, for example:

```json
{ "Rules": [{ "ID": "expire-komputer-sessions", "Status": "Enabled",
  "Filter": { "Prefix": "komputer/" }, "Expiration": { "Days": 90 } }] }
```

## Schedules and Kubernetes 1.30+

On Kubernetes 1.30 and later the chart installs a `ValidatingAdmissionPolicy` that reads `KomputerConfig` and rejects a schedule with `lifecycle: Archive` while the store is off — including schedules applied with `kubectl`. On older clusters the API still rejects them, and a schedule applied directly goes to `Error` when it runs.
