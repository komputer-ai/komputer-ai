"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { motion } from "framer-motion";
import { Ban, Trash2, Zap, Save, Check, Plus } from "lucide-react";
import { Button } from "@/components/kit/button";
import { Badge } from "@/components/kit/badge";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/kit/tabs";
import { StatusBadge } from "@/components/shared/status-badge";
import { CostBadge } from "@/components/shared/cost-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { Tooltip } from "@/components/kit/tooltip";
import { AgentChat } from "@/components/agents/agent-chat";
import { useWebSocket } from "@/hooks/use-websocket";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { getAgent, deleteAgent, cancelAgent, createAgent, getAgentEvents, patchAgent, listMemories, listSkills } from "@/lib/api";
import { SubAgentPanel } from "@/components/agents/sub-agent-panel";
import { MODELS, LIFECYCLES } from "@/lib/constants";
import { Input } from "@/components/kit/input";
import { Textarea } from "@/components/kit/textarea";
import { Label } from "@/components/kit/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/kit/select";
import type { AgentResponse, AgentEvent } from "@/lib/types";

export default function AgentDetailPage() {
  const params = useParams<{ name: string }>();
  const searchParams = useSearchParams();
  const router = useRouter();
  const agentName = params.name;
  const agentNs = searchParams.get("namespace") || undefined;
  const initialPending = searchParams.get("pending") || undefined;

  const [agent, setAgent] = useState<AgentResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const showLoading = useDelayedLoading(loading);
  const [error, setError] = useState<string | null>(null);

  const { events: wsEvents } = useWebSocket(agentName);
  const [historyEvents, setHistoryEvents] = useState<AgentEvent[]>([]);
  const [hasMoreEvents, setHasMoreEvents] = useState(true);
  const [loadingOlder, setLoadingOlder] = useState(false);

  const parseEventsResponse = useCallback((data: unknown): AgentEvent[] => {
    return Array.isArray(data) ? data : (data as { events?: AgentEvent[] })?.events ?? [];
  }, []);

  // Fetch event history on mount
  useEffect(() => {
    if (!agentName) return;
    getAgentEvents(agentName, 50, agentNs)
      .then((data: unknown) => {
        const arr = parseEventsResponse(data);
        setHistoryEvents(arr);
        if (arr.length < 50) setHasMoreEvents(false);
      })
      .catch(() => {});
  }, [agentName, agentNs, parseEventsResponse]);

  // Load older events (called when user scrolls to top)
  const loadOlderEvents = useCallback(async () => {
    if (!agentName || loadingOlder || !hasMoreEvents) return;
    const oldestTimestamp = historyEvents.length > 0 ? historyEvents[0].timestamp : undefined;
    if (!oldestTimestamp) return;
    setLoadingOlder(true);
    try {
      const data = await getAgentEvents(agentName, 50, agentNs, oldestTimestamp);
      const older = parseEventsResponse(data);
      if (older.length === 0) {
        setHasMoreEvents(false);
      } else {
        setHistoryEvents((prev) => [...older, ...prev]);
        if (older.length < 50) setHasMoreEvents(false);
      }
    } catch {
      // Silently fail, user can scroll up again to retry
    } finally {
      setLoadingOlder(false);
    }
  }, [agentName, agentNs, historyEvents, loadingOlder, hasMoreEvents, parseEventsResponse]);

  // Merge history + WS events, deduplicating by full fingerprint
  const events = useMemo(() => {
    const all = [...historyEvents, ...wsEvents];
    const seen = new Set<string>();
    return all
      .filter((e) => {
        const key = `${e.timestamp}:${e.type}:${e.payload?.content ?? e.payload?.text ?? e.payload?.message ?? e.payload?.instructions ?? ""}`;
        if (seen.has(key)) return false;
        seen.add(key);
        return true;
      })
      .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
  }, [historyEvents, wsEvents]);

  // Fetch agent info with polling
  const fetchAgent = useCallback(async () => {
    if (!agentName) return;
    try {
      const data = await getAgent(agentName, agentNs);
      setAgent(data);
      setError(null);
    } catch (e: unknown) {
      if (e instanceof Error && e.message.includes("not found")) {
        setError("Agent not found");
      } else {
        setError(e instanceof Error ? e.message : "Unknown error");
      }
    } finally {
      setLoading(false);
    }
  }, [agentName, agentNs]);

  useEffect(() => {
    fetchAgent();
  }, [fetchAgent]);

  // Poll only while status is transient (Pending, or Running with an active task)
  const needsPolling = agent != null && (
    agent.status === "Pending" ||
    agent.taskStatus === "InProgress"
  );
  useEffect(() => {
    if (!needsPolling) return;
    const interval = setInterval(fetchAgent, 5000);
    return () => clearInterval(interval);
  }, [needsPolling, fetchAgent]);

  // Refetch agent when a task completes so costs update
  const lastEventType = events.length > 0 ? events[events.length - 1].type : null;
  useEffect(() => {
    if (lastEventType === "task_completed" || lastEventType === "task_cancelled") {
      fetchAgent();
    }
  }, [lastEventType, fetchAgent]);

  // Actions
  const handleCancel = async () => {
    if (!agentName) return;
    try {
      await cancelAgent(agentName, agentNs);
      fetchAgent();
    } catch {
      // Will be reflected in next poll
    }
  };

  const handleDelete = async () => {
    if (!agentName) return;
    try {
      await deleteAgent(agentName, agentNs);
      router.push("/agents");
    } catch {
      // Will be reflected in next poll
    }
  };

  const handleWake = async () => {
    if (!agentName) return;
    try {
      await createAgent({ name: agentName, instructions: "wake" });
      fetchAgent();
    } catch {
      // Will be reflected in next poll
    }
  };

  // Loading state
  if (showLoading) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex flex-1 items-center justify-center text-sm text-[var(--color-text-secondary)]">
          Loading agent...
        </div>
      </div>
    );
  }

  // Still loading but delay hasn't elapsed yet — render nothing to avoid flash
  if (loading) {
    return null;
  }

  // Error / not found
  if (error || !agent) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex flex-1 items-center justify-center">
          <div className="rounded-lg border border-red-400/20 bg-red-400/5 p-6 text-center">
            <p className="text-sm text-red-400">{error ?? "Agent not found"}</p>
            <Button
              variant="ghost"
              size="sm"
              className="mt-3"
              onClick={() => router.push("/agents")}
            >
              Back to agents
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, ease: "easeOut" }}
      className="flex h-full flex-col"
    >
      {/* Agent info bar */}
      <div className="flex flex-wrap items-center gap-3 border-b border-[var(--color-border)] bg-[var(--color-bg)] px-6 py-3">
        <span className="text-sm text-[var(--color-text-muted)]">Status:</span>
        <StatusBadge status={agent.status} />
        {agent.taskStatus && (
          <>
            <span className="text-sm text-[var(--color-text-muted)]">Task:</span>
            <StatusBadge status={agent.taskStatus} size="sm" />
          </>
        )}
        <span className="text-sm text-[var(--color-text-muted)]">Model:</span>
        <Badge variant="outline" className="text-xs font-mono">
          {agent.model}
        </Badge>
        <span className="text-sm text-[var(--color-text-muted)]">Namespace:</span>
        <Tooltip content="Namespace" side="bottom">
          <span className="cursor-default text-sm text-[var(--color-text-secondary)]">
            {agent.namespace}
          </span>
        </Tooltip>
        {agent.lifecycle && (
          <Badge variant="secondary" className="text-xs">
            {agent.lifecycle}
          </Badge>
        )}

        <div className="ml-auto flex items-center gap-3">
          {/* Costs */}
          <div className="flex items-center gap-2 text-sm text-[var(--color-text-secondary)]">
            <span>Last:</span>
            <CostBadge cost={agent.lastTaskCostUSD} />
            <span>Total:</span>
            <CostBadge cost={agent.totalCostUSD} />
          </div>

          <span className="text-[var(--color-border)]">|</span>

          <RelativeTime timestamp={agent.createdAt} />

          <span className="text-[var(--color-border)]">|</span>

          {/* Action buttons */}
          <div className="flex items-center gap-1.5">
            {agent.status === "Sleeping" && (
              <Button variant="secondary" size="sm" onClick={handleWake}>
                <Zap className="size-3" data-icon="inline-start" />
                Wake
              </Button>
            )}
            {agent.taskStatus === "InProgress" && (
              <Button variant="secondary" size="sm" onClick={handleCancel}>
                <Ban className="size-3" data-icon="inline-start" />
                Cancel
              </Button>
            )}
            <ConfirmDialog
              title="Delete Agent"
              description={`Are you sure you want to delete "${agent.name}"? This action cannot be undone.`}
              onConfirm={handleDelete}
              trigger={
                <Button variant="destructive" size="sm">
                  <Trash2 className="size-3" data-icon="inline-start" />
                  Delete
                </Button>
              }
            />
          </div>
        </div>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="chat" className="flex flex-1 flex-col overflow-hidden">
        <div className="shrink-0 border-b border-[var(--color-border)] px-6">
          <TabsList>
            <TabsTrigger value="chat">Chat</TabsTrigger>
            <TabsTrigger value="info">Settings</TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="chat" className="flex-1 overflow-hidden flex">
          <AgentChat
            agentName={agent.name}
            agentNamespace={agentNs}
            agentStatus={agent.status}
            agentLifecycle={agent.lifecycle}
            events={events}
            taskStatus={agent.taskStatus}
            initialPending={agent.taskStatus === "Complete" || agent.taskStatus === "Error" ? undefined : initialPending}
            hasMoreEvents={hasMoreEvents}
            loadingOlder={loadingOlder}
            onLoadOlder={loadOlderEvents}
          />
          <SubAgentPanel agentName={agent.name} events={events} namespace={agentNs} />
        </TabsContent>

        <TabsContent value="info" className="flex-1 overflow-y-auto p-6">
          <div className="mx-auto max-w-4xl space-y-6">
            {/* Top stats row */}
            <motion.div
              className="grid grid-cols-2 sm:grid-cols-4 gap-3"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, ease: "easeOut" }}
            >
              <StatCard
                label="Phase"
                value={agent.status}
                color={
                  agent.status === "Running" ? "var(--color-status-running)" :
                  agent.status === "Sleeping" ? "var(--color-status-sleeping)" :
                  agent.status === "Failed" ? "var(--color-status-error)" :
                  agent.status === "Pending" ? "var(--color-status-pending)" :
                  agent.status === "Succeeded" ? "var(--color-status-success)" : undefined
                }
              />
              <StatCard label="Task" value={agent.taskStatus || "Idle"} />
              <StatCard label="Total Cost" value={agent.totalCostUSD ? `$${agent.totalCostUSD}` : "—"} highlight />
              <StatCard label="Last Task" value={agent.lastTaskCostUSD ? `$${agent.lastTaskCostUSD}` : "—"} />
            </motion.div>

            {/* Settings */}
            <SettingsCard agent={agent} agentNs={agentNs} onSaved={fetchAgent} />

            {/* Agent info */}
            <motion.div
              className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-5 space-y-3 transition-colors duration-150 hover:border-[var(--color-border-hover)] hover:bg-[var(--color-surface-hover)]"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, ease: "easeOut", delay: 0.1 }}
            >
              <h3 className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)]">Agent Info</h3>
              <InfoRow label="Name" value={agent.name} />
              <InfoRow label="Namespace" value={agent.namespace} />
              <InfoRow label="Created" value={new Date(agent.createdAt).toLocaleString()} />
            </motion.div>
          </div>
        </TabsContent>
      </Tabs>
    </motion.div>
  );
}

function StatCard({ label, value, color, highlight }: {
  label: string;
  value: string;
  color?: string;
  highlight?: boolean;
}) {
  return (
    <div className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-4 transition-colors duration-150 hover:border-[var(--color-border-hover)] hover:bg-[var(--color-surface-hover)]">
      <p className="text-[10px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)] mb-1.5">{label}</p>
      <p
        className={`text-lg font-semibold ${highlight ? "text-[var(--color-brand-blue)]" : "text-[var(--color-text)]"}`}
        style={color ? { color } : undefined}
      >
        {value}
      </p>
    </div>
  );
}

function SettingsCard({ agent, agentNs, onSaved }: {
  agent: AgentResponse;
  agentNs?: string;
  onSaved: () => void;
}) {
  const [model, setModel] = useState(agent.model);
  const [lifecycle, setLifecycle] = useState<string>(agent.lifecycle || "default");
  const instructions = agent.instructions ?? "";
  const [newSecrets, setNewSecrets] = useState<{ key: string; value: string }[]>([]);
  const [agentMemories, setAgentMemories] = useState<string[]>(agent.memories ?? []);
  const [availableMemories, setAvailableMemories] = useState<{ name: string; namespace: string; ref: string }[]>([]);
  const [agentSkills, setAgentSkills] = useState<string[]>(agent.skills ?? []);
  const [availableSkills, setAvailableSkills] = useState<{ name: string; namespace: string; ref: string }[]>([]);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listMemories().then((res) => {
      setAvailableMemories((res.memories ?? []).map((m) => ({
        name: m.name,
        namespace: m.namespace,
        ref: m.namespace === (agentNs || "default") ? m.name : `${m.namespace}/${m.name}`,
      })));
    }).catch(() => {});
    listSkills().then((res) => {
      setAvailableSkills((res.skills ?? []).map((s) => ({
        name: s.name,
        namespace: s.namespace,
        ref: s.namespace === (agentNs || "default") ? s.name : `${s.namespace}/${s.name}`,
      })));
    }).catch(() => {});
  }, [agentNs]);

  const agentLifecycle = agent.lifecycle || "default";
  const memoriesChanged = JSON.stringify(agentMemories.sort()) !== JSON.stringify((agent.memories ?? []).sort());
  const skillsChanged = JSON.stringify(agentSkills.sort()) !== JSON.stringify((agent.skills ?? []).sort());
  const hasChanges = model !== agent.model || lifecycle !== agentLifecycle || newSecrets.some(s => s.key.trim() && s.value.trim()) || memoriesChanged || skillsChanged;

  async function handleSave() {
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      const patch: Record<string, unknown> = {};
      if (model !== agent.model) patch.model = model;
      if (lifecycle !== agentLifecycle) patch.lifecycle = lifecycle === "default" ? "" : lifecycle;
      const secretsMap: Record<string, string> = {};
      for (const s of newSecrets) {
        const k = s.key.trim();
        const v = s.value.trim();
        if (k && v) secretsMap[k] = v;
      }
      if (Object.keys(secretsMap).length > 0) patch.secrets = secretsMap;
      if (memoriesChanged) patch.memories = agentMemories;
      if (skillsChanged) patch.skills = agentSkills;
      await patchAgent(agent.name, patch, agentNs);
      setNewSecrets([]);
      setSaved(true);
      onSaved();
      setTimeout(() => setSaved(false), 2000);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save");
    } finally {
      setSaving(false);
    }
  }

  return (
    <motion.div
      className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-5 space-y-4 transition-colors duration-150 hover:border-[var(--color-border-hover)]"
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, ease: "easeOut", delay: 0.2 }}
    >
      <h3 className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)]">Settings</h3>

      <div className="flex flex-col gap-1.5">
        <Label>Instructions</Label>
        <Textarea
          value={instructions}
          disabled
          placeholder="No instructions set"
          style={{ minHeight: 100 }}
          className="opacity-60 cursor-not-allowed"
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <Label>Model</Label>
        <Select value={model} onValueChange={(v) => v && setModel(v)}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {MODELS.map((m) => (
              <SelectItem key={m.value} value={m.value}>{m.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label>Lifecycle</Label>
        <Select value={lifecycle} onValueChange={(v) => v && setLifecycle(v)}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {LIFECYCLES.map((l) => (
              <SelectItem key={l.value} value={l.value}>{l.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label>Secrets</Label>
        {agent.secrets && agent.secrets.length > 0 && (
          <div className="space-y-1.5 mb-1">
            {agent.secrets.map((key) => (
              <div key={key} className="flex items-center gap-3 rounded-[var(--radius-sm)] bg-[var(--color-bg)] px-3 py-2">
                <span className="text-[12px] font-[family-name:var(--font-mono)] text-[var(--color-text-secondary)]">{key}</span>
                <span className="ml-auto font-[family-name:var(--font-mono)] text-[11px] tracking-widest text-[var(--color-text-muted)]">••••••</span>
              </div>
            ))}
          </div>
        )}
        {newSecrets.map((secret, index) => (
          <div key={index} className="flex items-center gap-2">
            <Input
              placeholder="KEY"
              value={secret.key}
              onChange={(e) => setNewSecrets(prev => prev.map((s, i) => i === index ? { ...s, key: e.target.value } : s))}
              autoComplete="off"
              className="flex-1"
            />
            <Input
              type="password"
              placeholder="value"
              value={secret.value}
              onChange={(e) => setNewSecrets(prev => prev.map((s, i) => i === index ? { ...s, value: e.target.value } : s))}
              autoComplete="off"
              className="flex-1"
            />
            <Button type="button" variant="ghost" size="icon" onClick={() => setNewSecrets(prev => prev.filter((_, i) => i !== index))} className="shrink-0">
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        ))}
        <Button type="button" variant="secondary" size="sm" onClick={() => setNewSecrets(prev => [...prev, { key: "", value: "" }])} className="w-fit">
          <Plus className="mr-1 h-4 w-4" />
          Add Secret
        </Button>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label>Memories</Label>
        <div className="flex flex-wrap gap-1.5">
          {availableMemories.map((m) => {
            const attached = agentMemories.includes(m.ref);
            const isCrossNs = m.ref.includes("/");
            return (
              <button
                key={m.ref}
                type="button"
                onClick={() => setAgentMemories(prev =>
                  attached ? prev.filter(n => n !== m.ref) : [...prev, m.ref]
                )}
                className={`text-xs px-2.5 py-1 rounded-full border transition-colors cursor-pointer ${
                  attached
                    ? "border-[var(--color-text)] bg-white/10 text-[var(--color-text)]"
                    : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
                }`}
              >
                {attached && <Check className="inline size-2.5 mr-1" />}
                {m.name}
                {isCrossNs && <span className="ml-1 text-[9px] text-[var(--color-brand-blue-light)]">{m.namespace}</span>}
              </button>
            );
          })}
          {availableMemories.length === 0 && (
            <p className="text-xs text-[var(--color-text-muted)]">No memories available</p>
          )}
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label>Skills</Label>
        <div className="flex flex-wrap gap-1.5">
          {availableSkills.map((s) => {
            const attached = agentSkills.includes(s.ref);
            const isCrossNs = s.ref.includes("/");
            return (
              <button
                key={s.ref}
                type="button"
                onClick={() => setAgentSkills(prev =>
                  attached ? prev.filter(n => n !== s.ref) : [...prev, s.ref]
                )}
                className={`text-xs px-2.5 py-1 rounded-full border transition-colors cursor-pointer ${
                  attached
                    ? "border-violet-400 bg-violet-400/10 text-violet-300"
                    : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
                }`}
              >
                {attached && <Check className="inline size-2.5 mr-1" />}
                {s.name}
                {isCrossNs && <span className="ml-1 text-[9px] text-[var(--color-brand-blue-light)]">{s.namespace}</span>}
              </button>
            );
          })}
          {availableSkills.length === 0 && (
            <p className="text-xs text-[var(--color-text-muted)]">No skills available</p>
          )}
        </div>
      </div>

      {error && <p className="text-sm text-red-400">{error}</p>}

      <div className="flex items-center gap-2">
        <Button size="sm" onClick={handleSave} disabled={!hasChanges || saving}>
          {saved ? (
            <><Check className="size-3 mr-1" /> Saved</>
          ) : saving ? (
            "Saving..."
          ) : (
            <><Save className="size-3 mr-1" /> Save Changes</>
          )}
        </Button>
      </div>
    </motion.div>
  );
}

function InfoRow({ label, value, mono, color, highlight }: {
  label: string;
  value: string;
  mono?: boolean;
  color?: string;
  highlight?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-[11px] uppercase tracking-wider text-[var(--color-text-muted)] shrink-0">{label}</span>
      <span
        className={`text-[13px] truncate text-right ${
          mono ? "font-[family-name:var(--font-mono)]" : ""
        } ${highlight ? "font-semibold text-[var(--color-brand-blue)]" : "text-[var(--color-text-secondary)]"}`}
        style={color ? { color } : undefined}
      >
        {value}
      </span>
    </div>
  );
}
