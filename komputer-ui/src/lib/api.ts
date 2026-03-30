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
} from './types';
import { getConfig } from './config';

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
  request<AgentListResponse>(`/agents${ns ? `?namespace=${ns}` : ''}`);

export const getAgent = (name: string, ns?: string) =>
  request<AgentResponse>(`/agents/${name}${ns ? `?namespace=${ns}` : ''}`);

export const createAgent = (data: CreateAgentRequest) =>
  request<AgentResponse>('/agents', { method: 'POST', body: JSON.stringify(data) });

export const deleteAgent = (name: string, ns?: string) =>
  request<void>(`/agents/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'DELETE' });

export const cancelAgent = (name: string, ns?: string) =>
  request<void>(`/agents/${name}/cancel${ns ? `?namespace=${ns}` : ''}`, { method: 'POST' });

export const getAgentEvents = (name: string, limit = 50, ns?: string) =>
  request<AgentEvent[]>(`/agents/${name}/events?limit=${limit}${ns ? `&namespace=${ns}` : ''}`);

// Offices
export const listOffices = (ns?: string) =>
  request<OfficeListResponse>(`/offices${ns ? `?namespace=${ns}` : ''}`);

export const getOffice = (name: string, ns?: string) =>
  request<OfficeResponse>(`/offices/${name}${ns ? `?namespace=${ns}` : ''}`);

export const deleteOffice = (name: string, ns?: string) =>
  request<void>(`/offices/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'DELETE' });

export const getOfficeEvents = (name: string, limit = 50, ns?: string) =>
  request<AgentEvent[]>(`/offices/${name}/events?limit=${limit}${ns ? `&namespace=${ns}` : ''}`);

// Schedules
export const listSchedules = (ns?: string) =>
  request<ScheduleListResponse>(`/schedules${ns ? `?namespace=${ns}` : ''}`);

export const getSchedule = (name: string, ns?: string) =>
  request<ScheduleResponse>(`/schedules/${name}${ns ? `?namespace=${ns}` : ''}`);

export const createSchedule = (data: CreateScheduleRequest) =>
  request<ScheduleResponse>('/schedules', { method: 'POST', body: JSON.stringify(data) });

export const deleteSchedule = (name: string, ns?: string) =>
  request<void>(`/schedules/${name}${ns ? `?namespace=${ns}` : ''}`, { method: 'DELETE' });

// Health
export const checkHealth = async () => {
  const res = await fetch(`${getConfig().apiUrl}/healthz`);
  return res.ok;
};
