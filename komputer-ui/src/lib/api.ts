import type {
  AgentResponse,
  AgentListResponse,
  OfficeResponse,
  OfficeListResponse,
  ScheduleResponse,
  ScheduleListResponse,
  PatchScheduleRequest,
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
  UpdateConnectorRequest,
  ConnectorTemplateListResponse,
  CostBreakdownResponse,
  Squad,
  SquadListResponse,
  CreateSquadRequest,
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
//
// Leaving `limit` unset returns every matching agent in one response (the
// backward-compatible default relied on by callers like useAgents()). Set
// `limit`/`offset` to page through results — the response then also carries
// `total`/`nextOffset`/`facets`.
export const listAgents = (
  ns?: string,
  opts?: {
    labelSelectors?: string[];
    status?: string;
    q?: string;
    sort?: string;
    limit?: number;
    offset?: number;
  },
) => {
  if (MOCK_EMPTY) return Promise.resolve({ agents: [] } as AgentListResponse);
  const params = new URLSearchParams();
  if (ns) params.set('namespace', ns);
  if (opts?.status) params.set('status', opts.status);
  if (opts?.q) params.set('q', opts.q);
  if (opts?.sort) params.set('sort', opts.sort);
  if (opts?.limit != null) params.set('limit', String(opts.limit));
  if (opts?.offset != null) params.set('offset', String(opts.offset));
  for (const l of opts?.labelSelectors ?? []) {
    params.append('label', l);
  }
  const qs = params.toString();
  return request<AgentListResponse>(`/agents${qs ? `?${qs}` : ''}`);
};

export const getAgent = (name: string, ns?: string) =>
  request<AgentResponse>(`/agents/${name}${ns ? `?namespace=${ns}` : ''}`);

export const createAgent = (data: CreateAgentRequest) =>
  request<AgentResponse>('/agents', { method: 'POST', body: JSON.stringify(data) });

export const deleteAgent = (name: string, ns?: string, opts?: { recreatePod?: boolean }) => {
  const params = new URLSearchParams();
  if (ns) params.set('namespace', ns);
  if (opts?.recreatePod) params.set('recreatePod', 'true');
  const qs = params.toString();
  return request<void>(`/agents/${name}${qs ? `?${qs}` : ''}`, { method: 'DELETE' });
};

export const cancelAgent = (name: string, ns?: string) =>
  request<void>(`/agents/${name}/cancel${ns ? `?namespace=${ns}` : ''}`, { method: 'POST' });

// wakeAgent wakes a sleeping agent idle, without running any task — unlike
// createAgent(), which always carries a task. A no-op (returns the agent's
// current state) if the agent is not sleeping.
export const wakeAgent = (name: string, ns?: string) =>
  request<AgentResponse>(`/agents/${name}/wake${ns ? `?namespace=${ns}` : ''}`, { method: 'POST' });

// sleepAgent sleeps an agent right now — pod deleted, PVC/workspace kept —
// regardless of sleepTTL or a prior idle wake, and WITHOUT changing the
// agent's lifecycle policy. Unlike patchAgent(name, { lifecycle: "Sleep" }),
// which only sets a policy the operator applies once a task completes (and
// which a prior wake or a sleepTTL can silently suppress), this always takes
// effect immediately. A no-op (returns the agent's current state) if already
// sleeping; throws (409) if a task is in progress, or (400) for an
// ephemeral-workspace agent.
export const sleepAgent = (name: string, ns?: string) =>
  request<AgentResponse>(`/agents/${name}/sleep${ns ? `?namespace=${ns}` : ''}`, { method: 'POST' });

export const compactAgent = (name: string, opts?: { instructions?: string; namespace?: string }) =>
  request<{ status: string; name: string }>(
    `/agents/${name}/compact${opts?.namespace ? `?namespace=${opts.namespace}` : ''}`,
    {
      method: 'POST',
      body: opts?.instructions ? JSON.stringify({ instructions: opts.instructions }) : undefined,
    }
  );

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

export const patchSchedule = (name: string, data: PatchScheduleRequest, ns?: string) =>
  request<ScheduleResponse>(`/schedules/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'PATCH', body: JSON.stringify(data) });

export const triggerSchedule = (name: string, ns?: string) =>
  request<{ status: string; name: string; agentName: string }>(
    `/schedules/${name}/trigger${ns ? `?namespace=${ns}` : ''}`,
    { method: 'POST' }
  );

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

export const updateConnector = (name: string, data: UpdateConnectorRequest) =>
  request<ConnectorResponse>(`/connectors/${name}`, { method: 'PATCH', body: JSON.stringify(data) });

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

// Squads
export const listSquads = (ns?: string) =>
  MOCK_EMPTY
    ? Promise.resolve({ squads: [] } as SquadListResponse)
    : request<SquadListResponse>(`/squads${ns ? `?namespace=${ns}` : ''}`);

export const getSquad = (name: string, ns?: string) =>
  request<Squad>(`/squads/${name}${ns ? `?namespace=${ns}` : ''}`);

export const createSquad = (data: CreateSquadRequest) =>
  request<Squad>('/squads', { method: 'POST', body: JSON.stringify(data) });

export const deleteSquad = (name: string, ns?: string) =>
  request<void>(`/squads/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'DELETE' });

export const addSquadMember = (
  squadName: string,
  ns: string,
  member: { ref?: { name: string; namespace?: string }; spec?: unknown; name?: string },
) =>
  request<Squad>(`/squads/${squadName}/members?namespace=${ns}`, {
    method: 'POST',
    body: JSON.stringify(member),
  });

export const removeSquadMember = (squadName: string, ns: string, agentName: string) =>
  request<Squad>(`/squads/${squadName}/members/${agentName}?namespace=${ns}`, { method: 'DELETE' });

export const breakUpSquad = (squadName: string, ns?: string) =>
  request<Squad>(`/squads/${squadName}/break-up${ns ? `?namespace=${ns}` : ''}`, { method: 'POST' });

// Health
export const listNamespaces = () =>
  request<{ namespaces: string[] }>('/namespaces');

export const checkHealth = async () => {
  if (MOCK_EMPTY) return true;
  const res = await fetch(`${getConfig().apiUrl}/healthz`);
  return res.ok;
};
