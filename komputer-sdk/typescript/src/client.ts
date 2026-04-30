/**
 * High-level convenience client for the komputer.ai API.
 *
 * Hand-maintained — do not overwrite with generated output.
 *
 * @example
 * const client = new KomputerClient("http://localhost:8080");
 * const agent = await client.createAgent({
 *   name: "my-agent",
 *   instructions: "Say hello",
 *   model: "claude-sonnet-4-6",
 * });
 */

import { Configuration, ResponseError } from "./runtime";
import { AgentsApi, ConnectorsApi, MemoriesApi, OfficesApi, SchedulesApi, SecretsApi, SkillsApi, TemplatesApi } from "./apis";
import type { CreateAgentRequest, CreateConnectorRequest, CreateMemoryRequest, CreateScheduleAgentSpec, CreateScheduleRequest, CreateSecretRequest, CreateSkillRequest, PatchAgentRequest, PatchMemoryRequest, PatchScheduleRequest, PatchSkillRequest, UpdateSecretRequest, V1PodSpec, V1alpha1StorageSpec } from "./models";
import { AgentEventStream } from "./watch";
import type { AgentEvent } from "./watch";
export type { AgentEvent } from "./watch";

export class KomputerClient {
  private _agents: AgentsApi;
  private _connectors: ConnectorsApi;
  private _memories: MemoriesApi;
  private _offices: OfficesApi;
  private _schedules: SchedulesApi;
  private _secrets: SecretsApi;
  private _skills: SkillsApi;
  private _templates: TemplatesApi;
  private _baseUrl: string;

  constructor(baseUrl: string = "http://localhost:8080") {
    this._baseUrl = baseUrl.replace(/\/$/, "");
    const config = new Configuration({ basePath: this._baseUrl + "/api/v1" });
    this._agents = new AgentsApi(config);
    this._connectors = new ConnectorsApi(config);
    this._memories = new MemoriesApi(config);
    this._offices = new OfficesApi(config);
    this._schedules = new SchedulesApi(config);
    this._secrets = new SecretsApi(config);
    this._skills = new SkillsApi(config);
    this._templates = new TemplatesApi(config);
  }

  // --- Agents ---

  async listAgents() {
    return this._agents.listAgents({});
  }

  async createAgent(params: { name: string; instructions: string; connectors?: string[]; lifecycle?: string; memories?: string[]; model?: string; namespace?: string; officeManager?: string; podSpec?: V1PodSpec; priority?: number; role?: string; secretRefs?: string[]; skills?: string[]; storage?: V1alpha1StorageSpec; systemPrompt?: string; templateRef?: string }) {
    try {
      return await this._agents.createAgent({ request: { connectors: params.connectors, instructions: params.instructions, lifecycle: params.lifecycle, memories: params.memories, model: params.model, name: params.name, namespace: params.namespace, officeManager: params.officeManager, podSpec: params.podSpec, priority: params.priority, role: params.role, secretRefs: params.secretRefs, skills: params.skills, storage: params.storage, systemPrompt: params.systemPrompt, templateRef: params.templateRef } });
    } catch (e) {
      if (e instanceof ResponseError && e.response.status === 409) {
        return this._agents.patchAgent({ name: params.name, request: { connectors: params.connectors, instructions: params.instructions, lifecycle: params.lifecycle, memories: params.memories, model: params.model, podSpec: params.podSpec, priority: params.priority, secretRefs: params.secretRefs, skills: params.skills, storage: params.storage, systemPrompt: params.systemPrompt, templateRef: params.templateRef } });
      }
      throw e;
    }
  }

  async getAgent(name: string) {
    return this._agents.getAgent({ name });
  }

  async patchAgent(params: { name: string; connectors?: string[]; instructions?: string; lifecycle?: string; memories?: string[]; model?: string; podSpec?: V1PodSpec; priority?: number; secretRefs?: string[]; skills?: string[]; storage?: V1alpha1StorageSpec; systemPrompt?: string; templateRef?: string }) {
    return this._agents.patchAgent({ name: params.name, request: { connectors: params.connectors, instructions: params.instructions, lifecycle: params.lifecycle, memories: params.memories, model: params.model, podSpec: params.podSpec, priority: params.priority, secretRefs: params.secretRefs, skills: params.skills, storage: params.storage, systemPrompt: params.systemPrompt, templateRef: params.templateRef } });
  }

  async deleteAgent(name: string) {
    return this._agents.deleteAgent({ name });
  }

  async cancelAgentTask(name: string) {
    return this._agents.cancelAgentTask({ name });
  }

  async getAgentEvents(name: string) {
    return this._agents.getAgentEvents({ name });
  }

  // --- Memories ---

  async listMemories() {
    return this._memories.listMemories({});
  }

  async createMemory(params: { name: string; content: string; description?: string; namespace?: string }) {
    try {
      return await this._memories.createMemory({ request: { content: params.content, description: params.description, name: params.name, namespace: params.namespace } });
    } catch (e) {
      if (e instanceof ResponseError && e.response.status === 409) {
        return this._memories.patchMemory({ name: params.name, request: { content: params.content, description: params.description } });
      }
      throw e;
    }
  }

  async getMemory(name: string) {
    return this._memories.getMemory({ name });
  }

  async patchMemory(params: { name: string; content?: string; description?: string }) {
    return this._memories.patchMemory({ name: params.name, request: { content: params.content, description: params.description } });
  }

  async deleteMemory(name: string) {
    return this._memories.deleteMemory({ name });
  }

  // --- Skills ---

  async listSkills() {
    return this._skills.listSkills({});
  }

  async createSkill(params: { name: string; content: string; description: string; namespace?: string }) {
    try {
      return await this._skills.createSkill({ request: { content: params.content, description: params.description, name: params.name, namespace: params.namespace } });
    } catch (e) {
      if (e instanceof ResponseError && e.response.status === 409) {
        return this._skills.patchSkill({ name: params.name, request: { content: params.content, description: params.description } });
      }
      throw e;
    }
  }

  async getSkill(name: string) {
    return this._skills.getSkill({ name });
  }

  async patchSkill(params: { name: string; content?: string; description?: string }) {
    return this._skills.patchSkill({ name: params.name, request: { content: params.content, description: params.description } });
  }

  async deleteSkill(name: string) {
    return this._skills.deleteSkill({ name });
  }

  // --- Schedules ---

  async listSchedules() {
    return this._schedules.listSchedules({});
  }

  async createSchedule(params: { name: string; instructions: string; schedule: string; agent?: CreateScheduleAgentSpec; agentName?: string; autoDelete?: boolean; keepAgents?: boolean; namespace?: string; timezone?: string }) {
    try {
      return await this._schedules.createSchedule({ request: { agent: params.agent, agentName: params.agentName, autoDelete: params.autoDelete, instructions: params.instructions, keepAgents: params.keepAgents, name: params.name, namespace: params.namespace, schedule: params.schedule, timezone: params.timezone } });
    } catch (e) {
      if (e instanceof ResponseError && e.response.status === 409) {
        return this._schedules.patchSchedule({ name: params.name, request: { schedule: params.schedule } });
      }
      throw e;
    }
  }

  async getSchedule(name: string) {
    return this._schedules.getSchedule({ name });
  }

  async patchSchedule(params: { name: string; schedule?: string }) {
    return this._schedules.patchSchedule({ name: params.name, request: { schedule: params.schedule } });
  }

  async deleteSchedule(name: string) {
    return this._schedules.deleteSchedule({ name });
  }

  // --- Secrets ---

  async listSecrets() {
    return this._secrets.listSecrets({});
  }

  async createSecret(params: { name: string; data: Record<string, string>; namespace?: string }) {
    try {
      return await this._secrets.createSecret({ request: { data: params.data, name: params.name, namespace: params.namespace } });
    } catch (e) {
      if (e instanceof ResponseError && e.response.status === 409) {
        return this._secrets.updateSecret({ name: params.name, request: { data: params.data, namespace: params.namespace } });
      }
      throw e;
    }
  }

  async updateSecret(params: { name: string; data: Record<string, string>; namespace?: string }) {
    return this._secrets.updateSecret({ name: params.name, request: { data: params.data, namespace: params.namespace } });
  }

  async deleteSecret(name: string) {
    return this._secrets.deleteSecret({ name });
  }

  // --- Connectors ---

  async listConnectors() {
    return this._connectors.listConnectors({});
  }

  async createConnector(params: { name: string; service: string; url: string; authSecretKey?: string; authSecretName?: string; authType?: string; displayName?: string; namespace?: string; oauthClientId?: string; oauthClientSecret?: string; type?: string }) {
    return this._connectors.createConnector({ request: { authSecretKey: params.authSecretKey, authSecretName: params.authSecretName, authType: params.authType, displayName: params.displayName, name: params.name, namespace: params.namespace, oauthClientId: params.oauthClientId, oauthClientSecret: params.oauthClientSecret, service: params.service, type: params.type, url: params.url } });
  }

  async getConnector(name: string) {
    return this._connectors.getConnector({ name });
  }

  async deleteConnector(name: string) {
    return this._connectors.deleteConnector({ name });
  }

  async listConnectorTools(name: string) {
    return this._connectors.listConnectorTools({ name });
  }

  // --- Offices ---

  async listOffices() {
    return this._offices.listOffices({});
  }

  async getOffice(name: string) {
    return this._offices.getOffice({ name });
  }

  async deleteOffice(name: string) {
    return this._offices.deleteOffice({ name });
  }

  async getOfficeEvents(name: string) {
    return this._offices.getOfficeEvents({ name });
  }

  // --- Templates ---

  async listTemplates() {
    return this._templates.listTemplates({});
  }


  // --- WebSocket ---

  /**
   * Stream live agent events.
   *
   * @param name Agent name.
   * @param options.group Optional consumer group name. When set, this watcher joins
   *   a group and each event is delivered to exactly one client per group across all
   *   API replicas — useful when running multiple SDK instances in a distributed system
   *   that should not each process the same event. Without `group`, every connected
   *   client receives every event (broadcast).
   */
  async watchAgent(name: string, options?: { group?: string }): Promise<AgentEventStream> {
    const wsUrl = this._baseUrl.replace("http://", "ws://").replace("https://", "wss://");
    let history: AgentEvent[] = [];
    try {
      const resp = await this._agents.getAgentEvents({ name, limit: 200 });
      if (resp && Array.isArray((resp as any).events)) {
        history = ((resp as any).events as any[]).map((e: any) => ({
          agentName: e.agentName || name,
          type: e.type || "",
          timestamp: e.timestamp || "",
          payload: e.payload || {},
        }));
      }
    } catch {
      // History fetch failed — proceed with live-only.
    }
    return new AgentEventStream(wsUrl, name, history, options?.group);
  }
}
