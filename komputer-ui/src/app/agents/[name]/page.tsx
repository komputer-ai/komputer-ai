"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { motion } from "framer-motion";
import { Ban, Trash2, Zap, KeyRound } from "lucide-react";

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
import { getAgent, deleteAgent, cancelAgent, createAgent, getAgentEvents } from "@/lib/api";
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

  // Fetch event history on mount
  useEffect(() => {
    if (!agentName) return;
    getAgentEvents(agentName, 50, agentNs)
      .then((data: unknown) => {
        const arr = Array.isArray(data) ? data : (data as { events?: AgentEvent[] })?.events ?? [];
        setHistoryEvents(arr);
      })
      .catch(() => {});
  }, [agentName, agentNs]);

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
      .sort((a, b) => a.timestamp.localeCompare(b.timestamp));
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
            <TabsTrigger value="info">Info</TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="chat" className="flex-1 overflow-hidden">
          <AgentChat
            agentName={agent.name}
            agentNamespace={agentNs}
            agentStatus={agent.status}
            events={events}
            taskStatus={agent.taskStatus}
            initialPending={initialPending}
          />
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

            {/* Details grid */}
            <motion.div
              className="grid grid-cols-1 md:grid-cols-2 gap-4"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, ease: "easeOut", delay: 0.1 }}
            >
              {/* Configuration */}
              <div className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-5 space-y-3 transition-colors duration-150 hover:border-[var(--color-border-hover)] hover:bg-[var(--color-surface-hover)]">
                <h3 className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)]">Configuration</h3>
                <InfoRow label="Name" value={agent.name} />
                <InfoRow label="Namespace" value={agent.namespace} />
                <InfoRow label="Model" value={agent.model} mono />
                <InfoRow label="Lifecycle" value={
                  agent.lifecycle === "AutoDelete" ? "Auto Delete" :
                  agent.lifecycle === "Sleep" ? "Sleep" : "Default"
                } />
                <InfoRow label="Created" value={new Date(agent.createdAt).toLocaleString()} />
              </div>

              {/* Secrets */}
              <div className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-5 space-y-3 transition-colors duration-150 hover:border-[var(--color-border-hover)] hover:bg-[var(--color-surface-hover)]">
                <div className="flex items-center gap-2">
                  <KeyRound className="size-3.5 text-[var(--color-text-muted)]" />
                  <h3 className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)]">Secrets</h3>
                </div>
                {agent.secrets && agent.secrets.length > 0 ? (
                  <div className="space-y-1.5">
                    {agent.secrets.map((key) => (
                      <div key={key} className="flex items-center gap-3 rounded-[var(--radius-sm)] bg-[var(--color-bg)] px-3 py-2 transition-colors duration-150 hover:bg-[var(--color-bg-subtle)]">
                        <span className="text-[12px] font-[family-name:var(--font-mono)] text-[var(--color-text-secondary)]">{key}</span>
                        <span className="ml-auto font-[family-name:var(--font-mono)] text-[11px] tracking-widest text-[var(--color-text-muted)]">••••••</span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-xs text-[var(--color-text-muted)]">No secrets configured</p>
                )}
              </div>
            </motion.div>

            {/* Last message */}
            {agent.lastTaskMessage && (
              <motion.div
                className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-5 space-y-2 transition-colors duration-150 hover:border-[var(--color-border-hover)] hover:bg-[var(--color-surface-hover)]"
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, ease: "easeOut", delay: 0.2 }}
              >
                <h3 className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)]">Last Task Message</h3>
                <p className="text-sm text-[var(--color-text-secondary)] leading-relaxed">{agent.lastTaskMessage}</p>
              </motion.div>
            )}
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
