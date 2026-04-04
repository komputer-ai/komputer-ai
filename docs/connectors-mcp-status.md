# MCP Connector Status

Services currently in the connector templates and their MCP authentication requirements.

## Ready (token-based, remote MCP)

| Service | MCP URL | Auth |
|---------|---------|------|
| GitHub | `https://api.githubcopilot.com/mcp/` | Personal Access Token (`ghp_`) |
| Gmail | `https://mcp.google.com/a/gmail/mcp` | Google OAuth Access Token (`ya29.`) |
| Google Calendar | `https://mcp.google.com/a/calendar/mcp` | Google OAuth Access Token (`ya29.`) |
| Linear | `https://mcp.linear.app/mcp` | API Key (`lin_api_`) |
| Slack | `https://mcp.slack.com/mcp` | User OAuth Token (`xoxp-`) |

> **Note:** Gmail and Google Calendar tokens are OAuth access tokens, but can be obtained manually via the [OAuth Playground](https://developers.google.com/oauthplayground) without a full OAuth flow in the UI.

---

## Needs OAuth (cannot use static token)

| Service | MCP URL | Reason |
|---------|---------|--------|
| Notion | `https://mcp.notion.com/mcp` | Requires OAuth 2.0 access token — integration tokens (`ntn_`) are rejected with `invalid_token` |
| Atlassian (Rovo) | `https://mcp.atlassian.com/v1/mcp` | PAT tokens only expose 2 limited tools (`getTeamworkGraphContext`, `getTeamworkGraphObject`) — full Jira/Confluence access requires OAuth |

---

## Needs Self-Hosted MCP Server

| Service | Recommended Server | Notes |
|---------|--------------------|-------|
| Atlassian (full) | [`sooperset/mcp-atlassian`](https://github.com/sooperset/mcp-atlassian) | Exposes ~72 Jira + Confluence tools via PAT — deploy in cluster, point custom connector at pod URL |
| Notion (full) | [`modelcontextprotocol/servers/notion`](https://github.com/modelcontextprotocol/servers/tree/main/src/notion) | Accepts `ntn_` integration tokens — deploy in cluster |
