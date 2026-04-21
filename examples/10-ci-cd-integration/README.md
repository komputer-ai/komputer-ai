# 10 — CI/CD Integration

Automatically debug failed GitHub Actions runs using a komputer.ai agent triggered by `workflow_run`.

## What it does

When your CI pipeline fails, the `debug-failed-run` workflow fires automatically. It:

1. Installs the `komputer` CLI
2. Runs an agent with the failed run ID and a `GITHUB_TOKEN` secret
3. The agent fetches the run details, downloads the failed step logs via the GitHub API, analyses the root cause, and posts a concrete fix suggestion — all streamed live to the workflow log

You can also trigger it manually from the Actions tab with any run ID.

## Setup

### 1. Add the secret

```
Settings → Secrets and variables → Actions → New repository secret
Name: KOMPUTER_API_URL
Value: http://your-komputer-api.example.com:8080
```

The `GITHUB_TOKEN` is provided automatically by Actions — no extra secret needed.

### 2. Copy the workflow

```bash
mkdir -p .github/workflows
cp .github/workflows/debug-failed-run.yml .github/workflows/
```

Edit the `workflows:` trigger to match your CI workflow name:

```yaml
on:
  workflow_run:
    workflows: ["CI"]   # ← change this to match your workflow's name field
    types: [completed]
```

### 3. Make the API reachable

The komputer.ai API must be reachable from GitHub Actions runners. Options:

- Public `LoadBalancer` service in your cluster
- Cloudflare Tunnel or similar
- A VPN/Tailscale exit node on the runner

## Trigger manually

```bash
# Trigger from the CLI with a specific run ID
gh workflow run debug-failed-run.yml -f run_id=12345678901

# Or from the GitHub UI: Actions → Debug Failed Run → Run workflow
```

## How the CLI simplifies this

The `komputer run` command creates the agent, streams all events to stdout, and exits when the task completes — in one line. No polling loop, no HTTP calls, no JSON parsing:

```bash
komputer run ci-debugger "Your task..." --secret GITHUB=$TOKEN --lifecycle AutoDelete
```

Compare this to the alternative: POST to create, poll `taskStatus` in a loop, then GET events to read the result. The CLI does all of that internally.

## Key concepts

- **`workflow_run`** — fires after another workflow completes; the `conclusion == 'failure'` condition ensures it only acts on failures
- **`--secret GITHUB=${{ secrets.GITHUB_TOKEN }}`** — injects the token as `SECRET_GITHUB` inside the agent pod; the agent uses it to call the GitHub API
- **`--lifecycle AutoDelete`** — the agent deletes itself after the task; no cleanup needed
- **`komputer run`** — create + stream + exit in one command, exit code reflects task success/failure so the workflow step passes or fails accordingly
