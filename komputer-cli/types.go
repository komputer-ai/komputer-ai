package main

// ─── API Types ───────────────────────────────────────────────────────────────

type AgentResponse struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	Model           string `json:"model"`
	Status          string `json:"status"`
	TaskStatus      string `json:"taskStatus"`
	LastTaskMessage string `json:"lastTaskMessage"`
	Lifecycle       string `json:"lifecycle"`
	LastTaskCostUSD string `json:"lastTaskCostUSD"`
	TotalCostUSD    string `json:"totalCostUSD"`
	CreatedAt       string `json:"createdAt"`
}

type AgentListResponse struct {
	Agents []AgentResponse `json:"agents"`
}

type AgentEvent struct {
	AgentName string                 `json:"agentName"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type OfficeResponse struct {
	Name            string                 `json:"name"`
	Namespace       string                 `json:"namespace"`
	Manager         string                 `json:"manager"`
	Phase           string                 `json:"phase"`
	TotalAgents     int                    `json:"totalAgents"`
	ActiveAgents    int                    `json:"activeAgents"`
	CompletedAgents int                    `json:"completedAgents"`
	TotalCostUSD    string                 `json:"totalCostUSD"`
	Members         []OfficeMemberResponse `json:"members"`
	CreatedAt       string                 `json:"createdAt"`
}

type OfficeMemberResponse struct {
	Name            string `json:"name"`
	Role            string `json:"role"`
	TaskStatus      string `json:"taskStatus"`
	LastTaskCostUSD string `json:"lastTaskCostUSD"`
}

type OfficeListResponse struct {
	Offices []OfficeResponse `json:"offices"`
}

type ScheduleResponse struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	Schedule       string `json:"schedule"`
	Timezone       string `json:"timezone"`
	AutoDelete     bool   `json:"autoDelete"`
	KeepAgents     bool   `json:"keepAgents"`
	Phase          string `json:"phase"`
	AgentName      string `json:"agentName"`
	NextRunTime    string `json:"nextRunTime"`
	LastRunTime    string `json:"lastRunTime"`
	RunCount       int    `json:"runCount"`
	SuccessfulRuns int    `json:"successfulRuns"`
	FailedRuns     int    `json:"failedRuns"`
	TotalCostUSD   string `json:"totalCostUSD"`
	LastRunCostUSD string `json:"lastRunCostUSD"`
	LastRunStatus  string `json:"lastRunStatus"`
	CreatedAt      string `json:"createdAt"`
}

type ScheduleListResponse struct {
	Schedules []ScheduleResponse `json:"schedules"`
}

type MemoryResponse struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Agents      []string `json:"agents"`
	CreatedAt   string   `json:"createdAt"`
}

type MemoryListResponse struct {
	Memories []MemoryResponse `json:"memories"`
}

type SkillResponse struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Agents      []string `json:"agentNames"`
	CreatedAt   string   `json:"createdAt"`
}

type SkillListResponse struct {
	Skills []SkillResponse `json:"skills"`
}

type ConnectorResponse struct {
	Name           string   `json:"name"`
	Namespace      string   `json:"namespace"`
	Service        string   `json:"service"`
	DisplayName    string   `json:"displayName"`
	URL            string   `json:"url"`
	Type           string   `json:"type"`
	AuthType       string   `json:"authType"`
	OAuthStatus    string   `json:"oauthStatus"`
	AuthSecretName string   `json:"authSecretName,omitempty"`
	AuthSecretKey  string   `json:"authSecretKey,omitempty"`
	AttachedAgents int      `json:"attachedAgents"`
	AgentNames     []string `json:"agentNames,omitempty"`
	CreatedAt      string   `json:"createdAt"`
}

type ConnectorListResponse struct {
	Connectors []ConnectorResponse `json:"connectors"`
}

type ConnectorToolResponse struct {
	Tools []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"tools"`
}

type ConnectorTemplateResponse struct {
	Service         string   `json:"service"`
	DisplayName     string   `json:"displayName"`
	Description     string   `json:"description"`
	URL             string   `json:"url"`
	AuthType        string   `json:"authType"`
	AuthLabel       string   `json:"authLabel"`
	AuthPlaceholder string   `json:"authPlaceholder"`
	GuideSteps      []string `json:"guideSteps"`
	Color           string   `json:"color"`
	LogoUrl         string   `json:"logoUrl"`
}

type ConnectorTemplateListResponse struct {
	Templates []ConnectorTemplateResponse `json:"templates"`
}
