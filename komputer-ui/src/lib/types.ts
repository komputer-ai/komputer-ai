export interface AgentResponse {
  name: string;
  namespace: string;
  model: string;
  status: 'Pending' | 'Running' | 'Sleeping' | 'Succeeded' | 'Failed';
  taskStatus?: 'InProgress' | 'Complete' | 'Error';
  lastTaskMessage?: string;
  lifecycle?: '' | 'Sleep' | 'AutoDelete';
  lastTaskCostUSD?: string;
  totalCostUSD?: string;
  totalTokens?: number;
  modelContextWindow?: number;
  secrets?: string[];
  memories?: string[];
  skills?: string[];
  connectors?: string[];
  instructions?: string;
  createdAt: string;
}

export interface AgentListResponse {
  agents: AgentResponse[];
}

export interface OfficeMemberResponse {
  name: string;
  role: 'manager' | 'worker';
  taskStatus?: string;
  lastTaskCostUSD?: string;
}

export interface OfficeResponse {
  name: string;
  namespace: string;
  manager: string;
  phase: 'InProgress' | 'Complete' | 'Error';
  totalAgents: number;
  activeAgents: number;
  completedAgents: number;
  totalCostUSD?: string;
  totalTokens?: number;
  members: OfficeMemberResponse[];
  createdAt: string;
}

export interface OfficeListResponse {
  offices: OfficeResponse[];
}

export interface ScheduleResponse {
  name: string;
  namespace: string;
  schedule: string;
  timezone?: string;
  autoDelete?: boolean;
  keepAgents?: boolean;
  phase: 'Active' | 'Suspended' | 'Error';
  agentName?: string;
  nextRunTime?: string;
  lastRunTime?: string;
  runCount?: number;
  successfulRuns?: number;
  failedRuns?: number;
  totalCostUSD?: string;
  lastRunCostUSD?: string;
  totalTokens?: number;
  lastRunTokens?: number;
  lastRunStatus?: string;
  createdAt: string;
}

export interface ScheduleListResponse {
  schedules: ScheduleResponse[];
}

export interface AgentEvent {
  agentName: string;
  namespace?: string;
  type: 'task_started' | 'user_message' | 'thinking' | 'tool_call' | 'tool_result' | 'text' | 'task_completed' | 'task_cancelled' | 'error';
  timestamp: string;
  payload: Record<string, any>;
}

export interface CreateAgentRequest {
  name: string;
  instructions: string;
  model?: string;
  templateRef?: string;
  role?: 'manager' | 'worker';
  namespace?: string;
  memories?: string[];
  skills?: string[];
  connectors?: string[];
  secretRefs?: string[];
  lifecycle?: '' | 'Sleep' | 'AutoDelete';
}

export interface CreateScheduleRequest {
  name: string;
  schedule: string;
  instructions: string;
  timezone?: string;
  autoDelete?: boolean;
  keepAgents?: boolean;
  agentName?: string;
  agent?: {
    model?: string;
    lifecycle?: string;
    role?: string;
  };
  namespace?: string;
}

export interface PatchAgentRequest {
  model?: string;
  lifecycle?: string;
  instructions?: string;
  templateRef?: string;
  secretRefs?: string[];
  memories?: string[];
  skills?: string[];
  connectors?: string[];
}

export interface ConnectorResponse {
  name: string;
  namespace: string;
  service: string;
  displayName: string;
  url: string;
  type: string;
  authSecretName?: string;
  authSecretKey?: string;
  authType?: string;
  oauthStatus?: string; // "pending" | "connected"
  attachedAgents: number;
  agentNames?: string[];
  createdAt: string;
}

export interface ConnectorListResponse {
  connectors: ConnectorResponse[];
}

export interface CreateConnectorRequest {
  name: string;
  service: string;
  displayName?: string;
  url: string;
  type?: string;
  authType?: string;
  authSecretName?: string;
  authSecretKey?: string;
  oauthClientId?: string;
  oauthClientSecret?: string;
  namespace?: string;
}

export interface TemplateResponse {
  name: string;
  scope: 'namespace' | 'cluster';
  namespace?: string;
}

export interface TemplateListResponse {
  templates: TemplateResponse[];
}

export interface MemoryResponse {
  name: string;
  namespace: string;
  content: string;
  description?: string;
  attachedAgents: number;
  agentNames?: string[];
  createdAt: string;
}

export interface MemoryListResponse {
  memories: MemoryResponse[];
}

export interface CreateMemoryRequest {
  name: string;
  content: string;
  description?: string;
  namespace?: string;
}

export interface SkillResponse {
  name: string;
  namespace: string;
  description: string;
  content: string;
  attachedAgents: number;
  agentNames?: string[];
  isDefault?: boolean;
  createdAt: string;
}

export interface SkillListResponse {
  skills: SkillResponse[];
}

export interface CreateSkillRequest {
  name: string;
  description: string;
  content: string;
  namespace?: string;
}

export interface SecretResponse {
  name: string;
  namespace: string;
  keys: string[];
  managed: boolean;
  agentName?: string;
  attachedAgents: number;
  agentNames?: string[];
  createdAt: string;
}

export interface SecretListResponse {
  secrets: SecretResponse[];
}

export interface CreateSecretRequest {
  name: string;
  data: Record<string, string>;
  namespace?: string;
}

export interface ConnectorTemplate {
  service: string;
  displayName: string;
  description: string;
  url: string;
  authType: "token" | "oauth" | "none";
  authLabel: string;
  authPlaceholder: string;
  guideSteps: string[];
  color: string;
  logoUrl: string;
  manifest?: string;
  manifestAfterStep?: number;
}

export interface ConnectorTemplateListResponse {
  templates: ConnectorTemplate[];
}
