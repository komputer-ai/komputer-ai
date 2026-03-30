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
  secrets?: string[];
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
  lastRunStatus?: string;
  createdAt: string;
}

export interface ScheduleListResponse {
  schedules: ScheduleResponse[];
}

export interface AgentEvent {
  agentName: string;
  namespace?: string;
  type: 'task_started' | 'thinking' | 'tool_call' | 'tool_result' | 'text' | 'task_completed' | 'task_cancelled' | 'error';
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
  secrets?: Record<string, string>;
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
