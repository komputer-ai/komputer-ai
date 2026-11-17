---
title: Authentication
description: Require a bearer token on the komputer API.
---

By default the API accepts unauthenticated requests, which is fine on a laptop or behind a private network. To require a token, turn on bearer auth.

## Turn it on

Create a Secret with the token, then point the chart at it:

```bash
kubectl create secret generic komputer-api-token \
  --from-literal=token="$(openssl rand -hex 32)" -n komputer-ai
```

```yaml
api:
  auth:
    enabled: true
    existingSecret: komputer-api-token
    secretKey: token
  cors:
    allowedOrigins:
      - https://komputer.example.com   # the UI's URL
```

The API then requires `Authorization: Bearer <token>` on every request. These stay open: `/healthz`, `/readyz`, `/api/metrics`, `/agent/metrics` and the OAuth callback (a browser redirect from the provider, protected by its `state` parameter).

## Who gets the token

| Caller | How |
|--------|-----|
| Operator | The chart passes it as `KOMPUTER_API_TOKEN`. |
| Manager agents | `KomputerConfig.spec.apiTokenSecret` names the Secret. The operator mirrors it into the agent's namespace and sets `KOMPUTER_API_TOKEN` on manager pods only, since workers don't call the API. |
| UI | Asks for the token the first time the API answers `401`, and keeps it in that browser's local storage. |
| CLI | `komputer login <endpoint> --token <token>`, or `KOMPUTER_API_TOKEN`. |
| MCP clients | Send `Authorization: Bearer <token>` to `/mcp`. |
| Your code | Send `Authorization: Bearer <token>`. |

## WebSockets

Browsers can't set headers on a WebSocket, so `/api/v1/agents/:name/ws` also accepts the token as `?access_token=<token>`. The query parameter is only accepted on WebSocket upgrades.

## CORS

`api.cors.allowedOrigins` limits which browser origins get CORS headers. Empty keeps the old behavior of allowing any origin. The API uses bearer tokens, not cookies, so this controls which sites can use a token a user pastes in, not whether a token leaks.

## Notes

- Rotate the token by updating the Secret and restarting the API and operator; manager agents pick it up on their next pod start.
- A single shared token gives full API access. Per-user or OIDC auth is a possible follow-up; until then, keep the API behind your network or an auth proxy as well.
