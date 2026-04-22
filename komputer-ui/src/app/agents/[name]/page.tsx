"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { Ban, Trash2, Zap, Moon, Save, Check, Plus, ChevronRight, Settings as SettingsIcon, MessageCircle, ArrowLeft } from "lucide-react";
import { CreateSecretModal } from "@/components/secrets/create-secret-modal";
import { Button } from "@/components/kit/button";
import { Badge } from "@/components/kit/badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { CostBadge } from "@/components/shared/cost-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { Tooltip } from "@/components/kit/tooltip";
import { AgentChat } from "@/components/agents/agent-chat";
import { useWebSocket } from "@/hooks/use-websocket";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { getAgent, deleteAgent, cancelAgent, createAgent, getAgentEvents, patchAgent, listMemories, listSkills, listSecrets, listConnectors } from "@/lib/api";
import { SubAgentPanel } from "@/components/agents/sub-agent-panel";
import { AgentTopology } from "@/components/agents/agent-topology";
import { MODELS, LIFECYCLES } from "@/lib/constants";
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

function fmtTokens(n: number): string {
  if (n >= 1_000_000) { const v = n / 1_000_000; return `${Number.isInteger(v) ? v : v.toFixed(1)}m`; }
  if (n >= 1000) { const v = n / 1000; return `${Number.isInteger(v) ? v : v.toFixed(1)}k`; }
  return String(n);
}

export default function AgentDetailPage() {
  const params = useParams<{ name: string }>();
  const searchParams = useSearchParams();
  const router = useRouter();
  const agentName = params.name;
  const agentNs = searchParams.get("namespace") || undefined;
  const initialPending = searchParams.get("pending") || undefined;
  // Capture task params on first mount only — refs prevent re-fetching when URL is cleaned up.
  const taskFromRef = useRef(searchParams.get("taskFrom") || undefined);
  const taskToRef = useRef(searchParams.get("taskTo") || undefined);
  const taskEventsRef = useRef(parseInt(searchParams.get("taskEvents") || "0", 10) || 0);
  const taskFrom = taskFromRef.current;
  const taskTo = taskToRef.current;
  const taskEvents = taskEventsRef.current;

  const [view, setView] = useState<"chat" | "settings">(searchParams.get("tab") === "info" ? "settings" : "chat");

  const setViewAndUrl = useCallback((next: "chat" | "settings") => {
    setView(next);
    const params = new URLSearchParams(window.location.search);
    if (next === "chat") { params.delete("tab"); } else { params.set("tab", "info"); }
    const qs = params.toString();
    window.history.replaceState(null, "", `${window.location.pathname}${qs ? `?${qs}` : ""}`);
  }, []);
  const [agent, setAgent] = useState<AgentResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const showLoading = useDelayedLoading(loading);
  const [error, setError] = useState<string | null>(null);
  const [sleeping, setSleeping] = useState(false);

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
    // When navigating from cost breakdown, fetch task events + 10 buffer on each side.
    const limit = taskFrom ? Math.max(taskEvents + 20, 50) : 50;
    getAgentEvents(agentName, limit, agentNs, undefined, undefined, taskFrom)
      .then((data: unknown) => {
        const arr = parseEventsResponse(data);
        setHistoryEvents(arr);
        if (!taskFrom && arr.length < 50) setHasMoreEvents(false);
      })
      .catch(() => {});
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agentName, agentNs, parseEventsResponse]);

  // Load older events (called when user scrolls to top)
  const historyEventsRef = useRef(historyEvents);
  historyEventsRef.current = historyEvents;
  const loadingOlderRef = useRef(false);
  const hasMoreEventsRef = useRef(hasMoreEvents);
  hasMoreEventsRef.current = hasMoreEvents;

  // Ref for the chat scroll container — set by AgentChat via callback.
  const scrollContainerRef = useRef<HTMLElement | null>(null);
  // Snapshot of scrollHeight taken BEFORE new messages are added to state.
  const scrollSnapshotRef = useRef<number | null>(null);

  const loadOlderEvents = useCallback(async () => {
    if (!agentName || loadingOlderRef.current || !hasMoreEventsRef.current) return;
    const oldest = historyEventsRef.current;
    const oldestTimestamp = oldest.length > 0 ? oldest[0].timestamp : undefined;
    if (!oldestTimestamp) return;
    loadingOlderRef.current = true;
    setLoadingOlder(true);
    try {
      const data = await getAgentEvents(agentName, 50, agentNs, oldestTimestamp);
      const older = parseEventsResponse(data);
      if (older.length === 0) {
        setHasMoreEvents(false);
      } else {
        // Snapshot scrollHeight BEFORE React updates the DOM.
        if (scrollContainerRef.current) {
          scrollSnapshotRef.current = scrollContainerRef.current.scrollHeight;
        }
        setHistoryEvents((prev) => [...older, ...prev]);
        if (older.length < 50) setHasMoreEvents(false);
      }
    } catch {
      // Silently fail, user can scroll up again to retry
    } finally {
      loadingOlderRef.current = false;
      setLoadingOlder(false);
    }
  }, [agentName, agentNs, parseEventsResponse]);

  const [hasNewerEvents, setHasNewerEvents] = useState(!!taskFrom);
  const [loadingNewer, setLoadingNewer] = useState(false);
  const loadingNewerRef = useRef(false);

  const loadNewerEvents = useCallback(async () => {
    if (!agentName || loadingNewerRef.current) return;
    const evts = historyEventsRef.current;
    const newestTimestamp = evts.length > 0 ? evts[evts.length - 1].timestamp : undefined;
    if (!newestTimestamp) return;
    loadingNewerRef.current = true;
    setLoadingNewer(true);
    try {
      const data = await getAgentEvents(agentName, 50, agentNs, undefined, undefined, undefined, newestTimestamp);
      const newer = parseEventsResponse(data);
      if (newer.length === 0) {
        setHasNewerEvents(false);
      } else {
        setHistoryEvents((prev) => [...prev, ...newer]);
        if (newer.length < 50) setHasNewerEvents(false);
      }
    } catch {
      // Silently fail
    } finally {
      loadingNewerRef.current = false;
      setLoadingNewer(false);
    }
  }, [agentName, agentNs, parseEventsResponse]);

  // Merge history + WS events, dedup by timestamp+type, sorted by time.
  // History events take precedence (listed first), so WS duplicates are dropped.
  const events = useMemo(() => {
    const seen = new Set<string>();
    return [...historyEvents, ...wsEvents]
      .filter((e) => {
        // Normalize user message types so task_started (WS) dedupes against user_message (history).
        const normType = e.type === "task_started" ? "user_message" : e.type;
        const key = `${e.timestamp}:${normType}`;
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
      if (data.status === "Sleeping") setSleeping(false);
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

  const handleSleep = async () => {
    if (!agentName) return;
    setSleeping(true);
    try {
      await patchAgent(agentName, { lifecycle: "Sleep" }, agentNs);
      await fetchAgent();
    } catch {
      // ignore
    } finally {
      setSleeping(false);
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
      {/* Unified header */}
      <div className="flex flex-1 flex-col overflow-hidden">
        <div className="shrink-0 border-b border-[var(--color-border)] bg-[var(--color-bg)] pl-6 pr-2">
          <div className="flex items-center gap-4 h-11">
            {/* Left: back button on settings view, otherwise status + model */}
            {view === "settings" ? (
              <Button variant="ghost" size="sm" onClick={() => setViewAndUrl("chat")}>
                <ArrowLeft className="size-3" data-icon="inline-start" />
                Back to chat
              </Button>
            ) : (
              <div className="flex items-center gap-3">
                <span className="text-[11px] text-[var(--color-text-muted)]">Status</span>
                <StatusBadge status={agent.status} />
                {agent.taskStatus && (
                  <>
                    <span className="text-[11px] text-[var(--color-text-muted)]">Task</span>
                    <StatusBadge status={agent.taskStatus} size="sm" />
                  </>
                )}
                <Badge variant="outline" className="text-xs font-mono">
                  {agent.model}
                </Badge>
                {agent.lifecycle && (
                  <Badge variant="secondary" className="text-xs">
                    {agent.lifecycle}
                  </Badge>
                )}
              </div>
            )}

            {/* Right: costs + actions */}
            <div className="ml-auto flex items-center gap-3">
              <div className="flex items-center gap-2 text-xs text-[var(--color-text-secondary)]">
                <span>Last</span>
                <CostBadge cost={agent.lastTaskCostUSD} />
                <span>Total</span>
                <CostBadge cost={agent.totalCostUSD} />
                {agent.totalTokens != null && agent.totalTokens > 0 && (
                  <>
                    <span className="text-[var(--color-border)]">·</span>
                    <span className="font-mono text-xs">
                      {fmtTokens(agent.totalTokens)} tokens
                    </span>
                  </>
                )}
              </div>

              <div className="h-4 w-px bg-[var(--color-border)]" />

              <RelativeTime timestamp={agent.createdAt} />

              <div className="h-4 w-px bg-[var(--color-border)]" />

              <div className="flex items-center gap-1.5">
                {agent.status === "Sleeping" ? (
                  <Button variant="secondary" size="sm" onClick={handleWake}>
                    <Zap className="size-3" data-icon="inline-start" />
                    Wake
                  </Button>
                ) : (
                  <>
                    {agent.taskStatus === "InProgress" && (
                      <Button variant="secondary" size="sm" onClick={handleCancel}>
                        <Ban className="size-3" data-icon="inline-start" />
                        Cancel
                      </Button>
                    )}
                    <Button variant="secondary" size="sm" onClick={handleSleep} disabled={sleeping}>
                      <Moon className={`size-3 ${sleeping ? "animate-spin" : ""}`} data-icon="inline-start" />
                      {sleeping ? "Sleeping…" : "Sleep"}
                    </Button>
                  </>
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
                <Tooltip
                  content={view === "settings" ? "Back to chat" : "Settings"}
                  side="bottom"
                >
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setViewAndUrl(view === "settings" ? "chat" : "settings")}
                    aria-label={view === "settings" ? "Back to chat" : "Settings"}
                    className="!px-0 w-7 !bg-white !text-[var(--color-bg)] hover:!bg-white/90 border-white/20"
                  >
                    <AnimatePresence mode="wait" initial={false}>
                      <motion.span
                        key={view}
                        className="inline-flex"
                        initial={{ opacity: 0, rotate: -90, scale: 0.7 }}
                        animate={{ opacity: 1, rotate: 0, scale: 1 }}
                        exit={{ opacity: 0, rotate: 90, scale: 0.7 }}
                        transition={{ duration: 0.18, ease: "easeOut" }}
                      >
                        {view === "settings" ? (
                          <MessageCircle className="size-4" />
                        ) : (
                          <SettingsIcon className="size-4" />
                        )}
                      </motion.span>
                    </AnimatePresence>
                  </Button>
                </Tooltip>
              </div>
            </div>
          </div>
        </div>

        {view === "chat" ? (
          <div className="flex-1 overflow-hidden flex">
            <AgentChat
              agentName={agent.name}
              agentNamespace={agentNs}
              agentStatus={agent.status}
              agentLifecycle={agent.lifecycle}
              agentContextWindow={agent.modelContextWindow}
              events={events}
              taskStatus={agent.taskStatus}
              initialPending={agent.taskStatus === "Complete" || agent.taskStatus === "Error" ? undefined : initialPending}
              hasMoreEvents={hasMoreEvents}
              loadingOlder={loadingOlder}
              onLoadOlder={loadOlderEvents}
              scrollContainerRef={scrollContainerRef}
              scrollSnapshotRef={scrollSnapshotRef}
              highlightTaskFrom={taskFrom}
              highlightTaskTo={taskTo}
              hasNewerEvents={hasNewerEvents}
              loadingNewer={loadingNewer}
              onLoadNewer={loadNewerEvents}
            />
            <SubAgentPanel agentName={agent.name} events={events} namespace={agentNs} />
          </div>
        ) : (
          <div className="flex-1 overflow-y-auto p-6">
            <div className="mx-auto max-w-4xl space-y-6">
              {/* Status + metrics */}
              <motion.div
                className="flex justify-center gap-3"
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, ease: "easeOut" }}
              >
                <StatusStrip status={agent.status} taskStatus={agent.taskStatus} />
                <MetricsRow
                  totalCostUSD={agent.totalCostUSD}
                  lastTaskCostUSD={agent.lastTaskCostUSD}
                  totalTokens={agent.totalTokens}
                />
              </motion.div>

              {/* Settings */}
              <SettingsCard agent={agent} agentNs={agentNs} onSaved={(updated) => { if (updated) setAgent(updated); fetchAgent(); }} />

              {/* Topology */}
              <AgentTopology agentName={agent.name} agentNs={agentNs} />

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
          </div>
        )}
      </div>
    </motion.div>
  );
}

function StatusStrip({ status, taskStatus }: { status: AgentResponse["status"]; taskStatus?: string }) {
  const accentColor =
    status === "Running" ? "var(--color-status-running)" :
    status === "Sleeping" ? "var(--color-status-sleeping)" :
    status === "Failed" ? "var(--color-status-error)" :
    status === "Pending" ? "var(--color-status-pending)" :
    status === "Succeeded" ? "var(--color-status-success)" :
    "var(--color-border)";

  const isRunning = status === "Running";

  return (
    <div
      className="relative flex items-center gap-6 rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] px-5 py-4 overflow-hidden"
      style={{ borderLeftColor: accentColor, borderLeftWidth: "3px" }}
    >
      {isRunning && (
        <div
          className="absolute left-0 top-0 bottom-0 w-16 pointer-events-none"
          style={{ background: `linear-gradient(to right, color-mix(in srgb, ${accentColor} 12%, transparent), transparent)` }}
        />
      )}
      <div className="flex items-center gap-3">
        <span className="text-[10px] uppercase tracking-widest font-semibold text-[var(--color-text-muted)]">Phase</span>
        <span className={`font-semibold text-sm ${isRunning ? "animate-pulse" : ""}`} style={{ color: accentColor }}>
          {status}
        </span>
      </div>
      <div className="w-px h-4 bg-[var(--color-border)]" />
      <div className="flex items-center gap-3">
        <span className="text-[10px] uppercase tracking-widest font-semibold text-[var(--color-text-muted)]">Task</span>
        <span className="font-semibold text-sm text-[var(--color-text)]">{taskStatus || "Idle"}</span>
      </div>
    </div>
  );
}

function MetricsRow({ totalCostUSD, lastTaskCostUSD, totalTokens }: {
  totalCostUSD?: string;
  lastTaskCostUSD?: string;
  totalTokens?: number;
}) {
  const items: { label: string; value: string; highlight?: boolean }[] = [
    { label: "Total Cost", value: totalCostUSD ? `$${totalCostUSD}` : "—", highlight: true },
    { label: "Last Task", value: lastTaskCostUSD ? `$${lastTaskCostUSD}` : "—" },
  ];
  if (totalTokens != null && totalTokens > 0) {
    items.push({
      label: "Total Tokens",
      value: fmtTokens(totalTokens),
    });
  }
  return (
    <div className="flex items-center rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] px-5 py-4 divide-x divide-[var(--color-border)]">
      {items.map((item) => (
        <div key={item.label} className="flex flex-col items-center gap-1 px-5 first:pl-0 last:pr-0">
          <span className="text-[10px] uppercase tracking-widest font-semibold text-[var(--color-text-muted)]">{item.label}</span>
          <span className={`font-mono font-semibold text-base tabular-nums ${item.highlight ? "text-[var(--color-brand-blue)]" : "text-[var(--color-text)]"}`}>
            {item.value}
          </span>
        </div>
      ))}
    </div>
  );
}

function SettingsCard({ agent, agentNs, onSaved }: {
  agent: AgentResponse;
  agentNs?: string;
  onSaved: (updated?: AgentResponse) => void;
}) {
  const [model, setModel] = useState(agent.model);
  const [lifecycle, setLifecycle] = useState<string>(agent.lifecycle || "default");
  const instructions = agent.instructions ?? "";
  const [systemPrompt, setSystemPrompt] = useState(agent.systemPrompt ?? "");
  const [systemPromptOpen, setSystemPromptOpen] = useState(!!agent.systemPrompt);
  const [agentSecretRefs, setAgentSecretRefs] = useState<string[]>(agent.secrets ?? []);
  const [availableSecrets, setAvailableSecrets] = useState<{ name: string; namespace: string }[]>([]);
  const [showAllSecrets, setShowAllSecrets] = useState(false);
  const [createSecretOpen, setCreateSecretOpen] = useState(false);
  const [agentMemories, setAgentMemories] = useState<string[]>(agent.memories ?? []);
  const [availableMemories, setAvailableMemories] = useState<{ name: string; namespace: string; ref: string }[]>([]);
  const [agentSkills, setAgentSkills] = useState<string[]>(agent.skills ?? []);
  const [availableSkills, setAvailableSkills] = useState<{ name: string; namespace: string; ref: string }[]>([]);
  const [agentConnectors, setAgentConnectors] = useState<string[]>(agent.connectors ?? []);
  const [availableConnectors, setAvailableConnectors] = useState<{ name: string; namespace: string; ref: string }[]>([]);
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
    listSecrets(agentNs || undefined, showAllSecrets).then((res) => {
      setAvailableSecrets((res.secrets ?? []).map((s) => ({ name: s.name, namespace: s.namespace })));
    }).catch(() => {});
    listConnectors().then((res) => {
      setAvailableConnectors((res.connectors ?? []).map((c) => ({
        name: c.name,
        namespace: c.namespace,
        ref: c.namespace === (agentNs || "default") ? c.name : `${c.namespace}/${c.name}`,
      })));
    }).catch(() => {});
  }, [agentNs, showAllSecrets]);

  const agentLifecycle = agent.lifecycle || "default";
  const memoriesChanged = JSON.stringify(agentMemories.sort()) !== JSON.stringify((agent.memories ?? []).sort());
  const skillsChanged = JSON.stringify(agentSkills.sort()) !== JSON.stringify((agent.skills ?? []).sort());
  const connectorsChanged = JSON.stringify(agentConnectors.sort()) !== JSON.stringify((agent.connectors ?? []).sort());
  const secretsChanged = JSON.stringify(agentSecretRefs.sort()) !== JSON.stringify((agent.secrets ?? []).sort());
  const systemPromptChanged = systemPrompt !== (agent.systemPrompt ?? "");
  const hasChanges = model !== agent.model || lifecycle !== agentLifecycle || secretsChanged || memoriesChanged || skillsChanged || connectorsChanged || systemPromptChanged;

  async function handleSave() {
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      const patch: Record<string, unknown> = {};
      if (model !== agent.model) patch.model = model;
      if (lifecycle !== agentLifecycle) patch.lifecycle = lifecycle === "default" ? "" : lifecycle;
      if (secretsChanged) patch.secretRefs = agentSecretRefs;
      if (memoriesChanged) patch.memories = agentMemories;
      if (skillsChanged) patch.skills = agentSkills;
      if (connectorsChanged) patch.connectors = agentConnectors;
      if (systemPromptChanged) patch.systemPrompt = systemPrompt.trim() || "";
      const updated = await patchAgent(agent.name, patch, agentNs);
      setSaved(true);
      onSaved(updated);
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

      {/* Collapsible system prompt */}
      <div className="flex flex-col">
        <button
          type="button"
          className="flex items-center gap-1.5 text-sm font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors cursor-pointer"
          onClick={() => setSystemPromptOpen(!systemPromptOpen)}
        >
          <ChevronRight className={`size-3.5 transition-transform duration-150 ${systemPromptOpen ? "rotate-90" : ""}`} />
          System Prompt <span className="text-[var(--color-text-muted)] font-normal">(Optional)</span>
        </button>
        <AnimatePresence initial={false}>
          {systemPromptOpen && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: "auto", opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.15, ease: "easeOut" }}
              className="overflow-hidden"
            >
              <div className="pt-2">
                <Textarea
                  value={systemPrompt}
                  onChange={(e) => setSystemPrompt(e.target.value)}
                  placeholder="Custom instructions that define agent behavior, persona, or constraints..."
                  style={{ minHeight: 100 }}
                />
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

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
        <div className="flex items-center justify-between">
          <Label>Secrets</Label>
          <button
            type="button"
            onClick={() => setShowAllSecrets((v) => !v)}
            className={`text-xs px-2 py-0.5 rounded-full border transition-colors cursor-pointer ${
              showAllSecrets
                ? "border-amber-500/50 bg-amber-500/10 text-amber-400"
                : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
            }`}
          >
            Show all
          </button>
        </div>
        <div className="flex flex-wrap gap-1.5">
          {availableSecrets.map((s) => {
            const attached = agentSecretRefs.includes(s.name);
            return (
              <button
                key={s.name}
                type="button"
                onClick={() => setAgentSecretRefs(prev =>
                  attached ? prev.filter(n => n !== s.name) : [...prev, s.name]
                )}
                className={`text-xs px-2.5 py-1 rounded-full border transition-colors cursor-pointer ${
                  attached
                    ? "border-[var(--color-text)] bg-white/10 text-[var(--color-text)]"
                    : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
                }`}
              >
                {attached && <Check className="inline size-2.5 mr-1" />}
                {s.name}
              </button>
            );
          })}
          <button
            type="button"
            onClick={() => setCreateSecretOpen(true)}
            className="text-xs px-2.5 py-1 rounded-full border border-dashed border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)] transition-colors cursor-pointer"
          >
            <Plus className="inline size-2.5 mr-1" />
            New Secret
          </button>
        </div>
        <CreateSecretModal
          open={createSecretOpen}
          onOpenChange={setCreateSecretOpen}
          onCreated={() => {
            listSecrets(agentNs || undefined, showAllSecrets).then((res) => {
              setAvailableSecrets((res.secrets ?? []).map((s) => ({ name: s.name, namespace: s.namespace })));
            }).catch(() => {});
          }}
        />
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
                    ? "border-[var(--color-text)] bg-white/10 text-[var(--color-text)]"
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

      <div className="flex flex-col gap-1.5">
        <Label>Connectors</Label>
        <div className="flex flex-wrap gap-1.5">
          {availableConnectors.map((c) => {
            const attached = agentConnectors.includes(c.ref);
            const isCrossNs = c.ref.includes("/");
            return (
              <button
                key={c.ref}
                type="button"
                onClick={() => setAgentConnectors(prev =>
                  attached ? prev.filter(n => n !== c.ref) : [...prev, c.ref]
                )}
                className={`text-xs px-2.5 py-1 rounded-full border transition-colors cursor-pointer ${
                  attached
                    ? "border-[var(--color-text)] bg-white/10 text-[var(--color-text)]"
                    : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
                }`}
              >
                {attached && <Check className="inline size-2.5 mr-1" />}
                {c.name}
                {isCrossNs && <span className="ml-1 text-[9px] text-[var(--color-brand-blue-light)]">{c.namespace}</span>}
              </button>
            );
          })}
          {availableConnectors.length === 0 && (
            <p className="text-xs text-[var(--color-text-muted)]">No connectors available</p>
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
