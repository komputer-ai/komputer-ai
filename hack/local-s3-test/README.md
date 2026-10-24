# Local test: S3 session store

An isolated kind cluster (`komputer-s3-test`, running in Docker Desktop's Docker)
with MinIO as the S3 store and komputer-ai built from this checkout. It has its
own kubeconfig, so your current kubectl context and the existing docker-desktop
install are never touched.

## Bring it up / tear it down

```bash
hack/local-s3-test/up.sh
```

```bash
SKIP_BUILD=1 hack/local-s3-test/up.sh
```

```bash
hack/local-s3-test/down.sh
```

`up.sh` builds the four images, loads them into kind, starts MinIO and creates
the `komputer-sessions` bucket, copies the Bedrock secret `default/komputer-aws-creds`
from the `docker-desktop` context (values are never printed), installs the Helm
chart with `values.yaml`, and adds NodePorts for host access.

After changing code: rebuild one image, `kind load docker-image <img>:s3-test --name komputer-s3-test`,
then `kubectl rollout restart` the deployment (agent pods pick up the new image on their next start).

## Where things are

| What | URL |
|---|---|
| API | http://localhost:28080/api/v1/config |
| UI | http://localhost:23000 |
| MinIO console | http://localhost:29001 (minioadmin / minioadmin-local) |
| MinIO S3 endpoint | http://localhost:29000 |

```bash
export KUBECONFIG=~/.kube/komputer-s3-test.yaml
```

Sessions land in the bucket at `komputer/<namespace>/<agent>/<uid>/session.tar.gz`.

## Automated scenarios

```bash
python3 hack/local-s3-test/e2e.py
```

Writes `hack/local-s3-test/RESULTS.md`. Uses Haiku on Bedrock (set `MODEL=…` to change).
`ONLY=2,3` runs a subset. Scenario 9 (an agent from before S3 was on gets saved from outside)
turns `s3.enabled` off and on again on the Helm release, so it only runs with `ONLY=9`.

## Things to try by hand

1. **Save after every task.** Create an agent in the UI, give it a task, open the
   agent page: the Session row shows when it was last saved. Check the object in the MinIO console.
2. **Archive and come back.** Create an agent with lifecycle *Archive*. After the
   task it disappears from the default list (filter by *Archived* to see it); its
   pod and PVC are gone. Send it another message: it wakes, restores its
   conversation from S3, answers, and archives again.
3. **Ephemeral + sleep.** Create an ephemeral agent (no PVC) with lifecycle *Sleep*.
   That's only allowed because S3 is on. Wake it and ask what you told it before.
4. **Chat history without Redis.** For an archived agent run
   `kubectl -n komputer-system exec deploy/komputer-ai-redis -- redis-cli DEL komputer-history:<agent>`
   (or restart Redis) and reload the agent page: the chat comes back from S3.
5. **A failed save stops the archive.** `kubectl -n minio scale deploy/minio --replicas=0`,
   run an Archive agent: the task ends in Error ("Couldn't save the session to S3: …"),
   the agent is not archived, and the UI offers *Delete without saving*. Scale MinIO back to 1.
6. **Delete waits for the save.** Delete an agent: it goes away once its session is
   saved; the object stays in the bucket afterwards.
7. **Schedules with Archive.** Create a schedule whose agent uses lifecycle *Archive*,
   trigger it twice: each run wakes, restores and re-archives the agent.
8. **Old agents get saved from outside.** Turn S3 off (`helm upgrade … --set s3.enabled=false`),
   create an agent and run a task, turn S3 back on: the operator copies the idle
   agent's session out to S3 within about a minute (condition `SessionSaved`).
