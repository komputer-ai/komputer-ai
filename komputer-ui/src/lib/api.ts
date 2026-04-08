import type {
  AgentResponse,
  AgentListResponse,
  OfficeResponse,
  OfficeListResponse,
  ScheduleResponse,
  ScheduleListResponse,
  CreateAgentRequest,
  CreateScheduleRequest,
  AgentEvent,
  PatchAgentRequest,
  TemplateListResponse,
  MemoryListResponse,
  CreateMemoryRequest,
  MemoryResponse,
  SkillListResponse,
  CreateSkillRequest,
  SkillResponse,
  SecretListResponse,
  SecretResponse,
  CreateSecretRequest,
  ConnectorListResponse,
  ConnectorResponse,
  CreateConnectorRequest,
  ConnectorTemplateListResponse,
  CostBreakdownResponse,
} from './types';
import { getConfig } from './config';

// Set to true to mock API responses with empty data (for UI development)
const MOCK_EMPTY = false;

function getBaseUrl() {
  return `${getConfig().apiUrl}/api/v1`;
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${getBaseUrl()}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

// Agents
export const listAgents = (ns?: string) =>
  MOCK_EMPTY
    ? Promise.resolve({ agents: [] } as AgentListResponse)
    : request<AgentListResponse>(`/agents${ns ? `?namespace=${ns}` : ''}`);

export const getAgent = (name: string, ns?: string) =>
  request<AgentResponse>(`/agents/${name}${ns ? `?namespace=${ns}` : ''}`);

export const createAgent = (data: CreateAgentRequest) =>
  request<AgentResponse>('/agents', { method: 'POST', body: JSON.stringify(data) });

export const deleteAgent = (name: string, ns?: string) =>
  request<void>(`/agents/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'DELETE' });

export const cancelAgent = (name: string, ns?: string) =>
  request<void>(`/agents/${name}/cancel${ns ? `?namespace=${ns}` : ''}`, { method: 'POST' });

export const getAgentEvents = (name: string, limit = 50, ns?: string, before?: string, source?: 'session' | 'redis', around?: string, after?: string) =>
  request<AgentEvent[]>(`/agents/${name}/events?limit=${limit}${ns ? `&namespace=${ns}` : ''}${before ? `&before=${encodeURIComponent(before)}` : ''}${source ? `&source=${source}` : ''}${around ? `&around=${encodeURIComponent(around)}` : ''}${after ? `&after=${encodeURIComponent(after)}` : ''}`);

export const getAgentCostBreakdown = (name: string, ns?: string, refresh?: boolean) => {
  const params = new URLSearchParams();
  if (ns) params.set("namespace", ns);
  if (refresh) params.set("refresh", "true");
  const qs = params.toString();
  return request<CostBreakdownResponse>(`/agents/${name}/cost${qs ? `?${qs}` : ''}`);
};

// Offices
export const listOffices = (ns?: string) =>
  MOCK_EMPTY
    ? Promise.resolve({ offices: [] } as OfficeListResponse)
    : request<OfficeListResponse>(`/offices${ns ? `?namespace=${ns}` : ''}`);

export const getOffice = (name: string, ns?: string) =>
  request<OfficeResponse>(`/offices/${name}${ns ? `?namespace=${ns}` : ''}`);

export const deleteOffice = (name: string, ns?: string) =>
  request<void>(`/offices/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'DELETE' });

export const getOfficeEvents = (name: string, limit = 50, ns?: string) =>
  request<AgentEvent[]>(`/offices/${name}/events?limit=${limit}${ns ? `&namespace=${ns}` : ''}`);

// Schedules
export const listSchedules = (ns?: string) =>
  MOCK_EMPTY
    ? Promise.resolve({ schedules: [] } as ScheduleListResponse)
    : request<ScheduleListResponse>(`/schedules${ns ? `?namespace=${ns}` : ''}`);

export const getSchedule = (name: string, ns?: string) =>
  request<ScheduleResponse>(`/schedules/${name}${ns ? `?namespace=${ns}` : ''}`);

export const createSchedule = (data: CreateScheduleRequest) =>
  request<ScheduleResponse>('/schedules', { method: 'POST', body: JSON.stringify(data) });

export const deleteSchedule = (name: string, ns?: string) =>
  request<void>(`/schedules/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'DELETE' });

export const patchSchedule = (name: string, data: { schedule?: string }, ns?: string) =>
  request<ScheduleResponse>(`/schedules/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'PATCH', body: JSON.stringify(data) });

// Agent settings
export const patchAgent = (name: string, data: PatchAgentRequest, ns?: string) =>
  request<AgentResponse>(`/agents/${name}${ns ? `?namespace=${ns}` : ''}`, {
    method: 'PATCH', body: JSON.stringify(data),
  });

// Memories
export const listMemories = (ns?: string) =>
  request<MemoryListResponse>(`/memories${ns ? `?namespace=${ns}` : ''}`);

export const getMemory = (name: string, ns?: string) =>
  request<MemoryResponse>(`/memories/${name}${ns ? `?namespace=${ns}` : ''}`);

export const createMemory = (data: CreateMemoryRequest) =>
  request<MemoryResponse>('/memories', { method: 'POST', body: JSON.stringify(data) });

export const deleteMemory = (name: string, ns?: string) =>
  request<void>(`/memories/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'DELETE' });

export const patchMemory = (name: string, data: { content?: string; description?: string }, ns?: string) =>
  request<MemoryResponse>(`/memories/${name}${ns ? `?namespace=${ns}` : ''}`, {
    method: 'PATCH', body: JSON.stringify(data),
  });

// Skills
export const listSkills = (ns?: string) =>
  request<SkillListResponse>(`/skills${ns ? `?namespace=${ns}` : ''}`);

export const getSkill = (name: string, ns?: string) =>
  request<SkillResponse>(`/skills/${name}${ns ? `?namespace=${ns}` : ''}`);

export const createSkill = (data: CreateSkillRequest) =>
  request<SkillResponse>('/skills', { method: 'POST', body: JSON.stringify(data) });

export const deleteSkill = (name: string, ns?: string) =>
  request<void>(`/skills/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'DELETE' });

export const patchSkill = (name: string, data: { content?: string; description?: string }, ns?: string) =>
  request<SkillResponse>(`/skills/${name}${ns ? `?namespace=${ns}` : ''}`, {
    method: 'PATCH', body: JSON.stringify(data),
  });

// Secrets
export const listSecrets = (ns?: string, all?: boolean) => {
  const params = new URLSearchParams();
  if (ns) params.set('namespace', ns);
  if (all) params.set('all', 'true');
  const qs = params.toString();
  return request<SecretListResponse>(`/secrets${qs ? `?${qs}` : ''}`);
};

export const createSecretResource = (data: CreateSecretRequest) =>
  request<SecretResponse>('/secrets', { method: 'POST', body: JSON.stringify(data) });

export const deleteSecretResource = (name: string, ns?: string) =>
  request<void>(`/secrets/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'DELETE' });

export const updateSecretResource = (name: string, data: Record<string, string>, ns?: string) =>
  request<void>(`/secrets/${name}`, { method: 'PATCH', body: JSON.stringify({ data, namespace: ns }) });

// Connectors
export const listConnectors = (ns?: string) =>
  request<ConnectorListResponse>(`/connectors${ns ? `?namespace=${ns}` : ''}`);

export const getConnector = (name: string, ns?: string) =>
  request<ConnectorResponse>(`/connectors/${name}${ns ? `?namespace=${ns}` : ''}`);

export const createConnector = (data: CreateConnectorRequest) =>
  request<ConnectorResponse>('/connectors', { method: 'POST', body: JSON.stringify(data) });

export const deleteConnector = (name: string, ns?: string) =>
  request<void>(`/connectors/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'DELETE' });

export const getConnectorTools = (name: string, ns?: string) =>
  request<{ tools: { name: string; description: string }[] }>(`/connectors/${name}/tools${ns ? `?namespace=${ns}` : ''}`);

export const getOAuthAuthorizeUrl = (data: {
  service: string;
  connector_name: string;
  displayName?: string;
  url?: string;
  oauthClientId: string;
  oauthClientSecret: string;
  namespace?: string;
}) =>
  request<{ authorizeUrl: string }>('/oauth/authorize', {
    method: 'POST',
    body: JSON.stringify(data),
  });

// Templates
export const listTemplates = (ns?: string) =>
  request<TemplateListResponse>(`/templates${ns ? `?namespace=${ns}` : ''}`);

// Connector templates
export const listConnectorTemplates = () =>
  request<ConnectorTemplateListResponse>('/connector-templates');

// Health
export const listNamespaces = () =>
  request<{ namespaces: string[] }>('/namespaces');

export const checkHealth = async () => {
  if (MOCK_EMPTY) return true;
  const res = await fetch(`${getConfig().apiUrl}/healthz`);
  return res.ok;
};
