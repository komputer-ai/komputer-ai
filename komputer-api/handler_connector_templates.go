package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ConnectorTemplateResponse describes a connector template served by GET /api/v1/connector-templates.
// It mirrors the ConnectorTemplate type in the UI's connector-templates.tsx.
type ConnectorTemplateResponse struct {
	Service           string   `json:"service"`
	DisplayName       string   `json:"displayName"`
	Description       string   `json:"description"`
	URL               string   `json:"url"`
	AuthType          string   `json:"authType"`
	AuthLabel         string   `json:"authLabel"`
	AuthPlaceholder   string   `json:"authPlaceholder"`
	GuideSteps        []string `json:"guideSteps"`
	Color             string   `json:"color"`
	LogoUrl           string   `json:"logoUrl"`
	Manifest          string   `json:"manifest,omitempty"`
	ManifestAfterStep *int     `json:"manifestAfterStep,omitempty"`
}

func intPtr(i int) *int { return &i }

// connectorTemplates is the canonical registry of all supported connector templates.
// Kept in sync with komputer-ui/src/lib/connector-templates.tsx.
var connectorTemplates = []ConnectorTemplateResponse{
	{
		Service:         "github",
		DisplayName:     "GitHub",
		Description:     "Repositories, issues, PRs, and code search.",
		URL:             "https://api.githubcopilot.com/mcp/",
		AuthType:        "token",
		AuthLabel:       "Personal Access Token",
		AuthPlaceholder: "ghp_xxxxxxxxxxxxxxxxxxxx",
		GuideSteps: []string{
			"Go to github.com/settings/tokens",
			"Click \"Generate new token (classic)\"",
			"Give it a name and select scopes (repo, issues, etc.)",
			"Click \"Generate token\" and copy it",
		},
		Color:   "#E5E7EB",
		LogoUrl: "https://cdn.simpleicons.org/github/white",
	},
	{
		Service:         "atlassian",
		DisplayName:     "Atlassian",
		Description:     "Jira issues and Confluence pages via Rovo MCP.",
		URL:             "https://mcp.atlassian.com/v1/mcp",
		AuthType:        "token",
		AuthLabel:       "API Token",
		AuthPlaceholder: "ATATT3xFfGF0...",
		GuideSteps: []string{
			"Make sure your org admin has enabled API token auth at support.atlassian.com/security-and-access-policies/docs/control-atlassian-rovo-mcp-server-settings",
			"Go to id.atlassian.com/manage-profile/security/api-tokens",
			"Click \"Create API token\", give it a label, and click \"Create\"",
			"Copy the token — it works for Jira, Confluence, and Compass",
		},
		Color:   "#2684FF",
		LogoUrl: "https://cdn.simpleicons.org/atlassian/2684FF",
	},
	{
		Service:         "slack",
		DisplayName:     "Slack",
		Description:     "Channels, messages, and threads.",
		URL:             "https://mcp.slack.com/mcp",
		AuthType:        "token",
		AuthLabel:       "User Token",
		AuthPlaceholder: "xoxp-xxxxxxxxxxxx-xxxxxxxxxxxx",
		GuideSteps: []string{
			"Go to api.slack.com/apps and click \"Create New App\"",
			"Choose \"From a manifest\", select your workspace, and paste the manifest below",
			"Click \"Create\", then go to \"Install App\" and install it to your workspace",
			"Go to Features → Agents & AI Apps and enable \"Model Context Protocol\"",
			"Copy the User OAuth Token from OAuth & Permissions (starts with xoxp-)",
		},
		ManifestAfterStep: intPtr(1),
		Manifest: `{
  "display_information": {
    "name": "Komputer AI"
  },
  "oauth_config": {
    "scopes": {
      "user": [
        "bookmarks:read",
        "bookmarks:write",
        "calls:read",
        "calls:write",
        "canvases:read",
        "canvases:write",
        "channels:history",
        "channels:read",
        "channels:write",
        "channels:write.invites",
        "channels:write.topic",
        "chat:write",
        "dnd:read",
        "dnd:write",
        "emoji:read",
        "files:read",
        "files:write",
        "groups:history",
        "groups:read",
        "groups:write",
        "groups:write.invites",
        "groups:write.topic",
        "im:history",
        "im:read",
        "im:write",
        "im:write.topic",
        "links:read",
        "links:write",
        "lists:read",
        "lists:write",
        "mpim:history",
        "mpim:read",
        "mpim:write",
        "mpim:write.topic",
        "pins:read",
        "pins:write",
        "reactions:read",
        "reactions:write",
        "reminders:read",
        "reminders:write",
        "remote_files:read",
        "remote_files:share",
        "search:read",
        "search:read.files",
        "search:read.im",
        "search:read.mpim",
        "search:read.private",
        "search:read.public",
        "search:read.users",
        "stars:read",
        "stars:write",
        "team:read",
        "team.preferences:read",
        "usergroups:read",
        "usergroups:write",
        "users.profile:read",
        "users.profile:write",
        "users:read",
        "users:read.email",
        "users:write"
      ]
    }
  },
  "settings": {
    "org_deploy_enabled": false,
    "socket_mode_enabled": false,
    "token_rotation_enabled": false
  }
}`,
		Color:   "#E01E5A",
		LogoUrl: "https://a.slack-edge.com/80588/marketing/img/icons/icon_slack_hash_colored.png",
	},
	{
		Service:         "gmail",
		DisplayName:     "Gmail",
		Description:     "Read, search, and draft emails.",
		URL:             "",
		AuthType:        "oauth",
		AuthLabel:       "OAuth Credentials",
		AuthPlaceholder: "",
		GuideSteps: []string{
			"Go to console.cloud.google.com/apis/credentials and create an OAuth 2.0 Client ID (type: Web application)",
			"Add your callback URL as an Authorized redirect URI (e.g. https://your-domain/api/v1/oauth/callback)",
			"Enable the Gmail API at console.cloud.google.com/apis/library/gmail.googleapis.com",
			"Copy the Client ID and Client Secret into the fields below, then click Connect",
		},
		Color:   "#EA4335",
		LogoUrl: "https://upload.wikimedia.org/wikipedia/commons/7/7e/Gmail_icon_%282020%29.svg",
	},
	{
		Service:         "google-calendar",
		DisplayName:     "Google Calendar",
		Description:     "Events, schedules, and availability.",
		URL:             "",
		AuthType:        "oauth",
		AuthLabel:       "OAuth Credentials",
		AuthPlaceholder: "",
		GuideSteps: []string{
			"Go to console.cloud.google.com/apis/credentials and create an OAuth 2.0 Client ID (type: Web application)",
			"Add your callback URL as an Authorized redirect URI (e.g. https://your-domain/api/v1/oauth/callback)",
			"Enable the Calendar API at console.cloud.google.com/apis/library/calendar-json.googleapis.com",
			"Copy the Client ID and Client Secret into the fields below, then click Connect",
		},
		Color:   "#4285F4",
		LogoUrl: "https://upload.wikimedia.org/wikipedia/commons/a/a5/Google_Calendar_icon_%282020%29.svg",
	},
	{
		Service:         "linear",
		DisplayName:     "Linear",
		Description:     "Issues, projects, and cycles.",
		URL:             "https://mcp.linear.app/mcp",
		AuthType:        "token",
		AuthLabel:       "API Key",
		AuthPlaceholder: "lin_api_xxxxxxxxxxxxxxxxxxxx",
		GuideSteps: []string{
			"Go to linear.app/settings/account/security",
			"Scroll to \"API keys\" and click \"Create key\"",
			"Give it a label and copy the key",
		},
		Color:   "#5E6AD2",
		LogoUrl: "https://cdn.simpleicons.org/linear/5E6AD2",
	},
	{
		Service:         "notion",
		DisplayName:     "Notion",
		Description:     "Pages, databases, and content.",
		URL:             "https://mcp.notion.com/mcp",
		AuthType:        "oauth",
		AuthLabel:       "OAuth Credentials",
		AuthPlaceholder: "",
		GuideSteps: []string{
			"Click Connect below to authorize access to your Notion workspace.",
			"Select the workspace and pages you want to share with your agents.",
		},
		Color:   "#E5E7EB",
		LogoUrl: "https://cdn.simpleicons.org/notion/white",
	},
	{
		Service:         "excalidraw",
		DisplayName:     "Excalidraw",
		Description:     "Hand-drawn diagrams and whiteboard sketches.",
		URL:             "https://mcp.excalidraw.com/mcp",
		AuthType:        "none",
		AuthLabel:       "",
		AuthPlaceholder: "",
		GuideSteps: []string{
			"No authentication required — just click Connect.",
		},
		Color:   "#6965DB",
		LogoUrl: "https://cdn.simpleicons.org/excalidraw/6965DB",
	},
	{
		Service:         "custom",
		DisplayName:     "Custom",
		Description:     "Connect any MCP server by URL.",
		URL:             "__custom__",
		AuthType:        "token",
		AuthLabel:       "Auth Token",
		AuthPlaceholder: "",
		GuideSteps:      []string{},
		Color:           "#6B7280",
		LogoUrl:         "",
	},
}

// listConnectorTemplates handles GET /api/v1/connector-templates.
func listConnectorTemplates() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"templates": connectorTemplates})
	}
}

// getConnectorTemplate returns the template for a given service, or nil if not found.
func getConnectorTemplate(service string) *ConnectorTemplateResponse {
	for i := range connectorTemplates {
		if connectorTemplates[i].Service == service {
			return &connectorTemplates[i]
		}
	}
	return nil
}
