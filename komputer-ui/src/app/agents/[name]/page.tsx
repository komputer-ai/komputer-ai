"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { Ban, Trash2, Zap, Moon, Archive, Plus, Settings as SettingsIcon, MessageCircle, ArrowLeft, Users, Layers, AlertTriangle, Hourglass, Timer, Eraser, Wallet, Loader2, CalendarDays, CloudOff, FileText, Cpu, Clock, Plug, Tag } from "lucide-react";
import Link from "next/link";
import { CreateSecretModal } from "@/components/secrets/create-secret-modal";
import { Button } from "@/components/kit/button";
import { Badge } from "@/components/kit/badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { SquadAwareDeleteDialog } from "@/components/shared/squad-aware-delete-dialog";
import { Tooltip } from "@/components/kit/tooltip";
import { AgentChat } from "@/components/agents/agent-chat";
import { SquadChatView } from "@/components/agents/squad-chat-view";
import { useWebSocket } from "@/hooks/use-websocket";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { getAgent, deleteAgent, cancelAgent, compactAgent, wakeAgent, sleepAgent, archiveAgent, getAgentEvents, patchAgent, listMemories, listSkills, listSecrets, listConnectors, deleteSquad } from "@/lib/api";
import { useSquads } from "@/hooks/use-squads";
import { TeamUpDialog } from "@/components/agents/team-up-dialog";
import { LeaveSquadButton } from "@/components/squads/leave-squad-button";
import type { Squad } from "@/lib/types";
import { SubAgentPanel } from "@/components/agents/sub-agent-panel";
import { AgentTopology } from "@/components/agents/agent-topology";
import { DetailsSection, RailSection, Field, PromptEditor, SaveBar, fmtIn } from "@/components/agents/agent-details-parts";
import { LIFECYCLES, EFFORTS } from "@/lib/constants";
import { Input } from "@/components/kit/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/kit/select";
import { ModelSelector } from "@/components/shared/model-selector";
import { MultiSelect, type MultiSelectOption } from "@/components/kit/multi-select";
import type { AgentResponse, AgentEvent, AgentCondition } from "@/lib/types";
import { cn, fmtDuration, formatRelativeTime } from "@/lib/utils";
import { humanizeConditionReason } from "@/lib/agent-conditions";
import { LabelsEditor } from "@/components/agents/labels-editor";
import { splitLabels, validateLabelRows, hasLabelErrors, diffLabels, type LabelRow } from "@/lib/label-edit";
import { usePlatformConfig } from "@/hooks/use-platform-config";
import { archivedReadOnly, agentDormant, ephemeralBlocksSleep, formatSessionStore } from "@/lib/session-store";

function fmtTokens(n: number): string {
  if (n >= 1_000_000) { const v = n / 1_000_000; return `${Number.isInteger(v) ? v : v.toFixed(1)}m`; }
  if (n >= 1000) { const v = n / 1000; return `${Number.isInteger(v) ? v : v.toFixed(1)}k`; }
  return String(n);
}

// Tooltip for a TTL badge: the absolute time the transition is due. An absent
// timestamp means that TTL's countdown isn't currently running (e.g. mid-task).
function ttlTitle(verb: string, expiresAt?: string): string {
  if (!expiresAt) return `${verb} once idle past this TTL`;
  const t = new Date(expiresAt);
  if (Number.isNaN(t.getTime())) return `${verb} once idle past this TTL`;
  return `${verb} at ${t.toLocaleString()}`;
}

// A dollar figure for the header: two decimals in the row, four in the tooltip.
function fmtUSD(cost: string | undefined, digits: number): string {
  const n = cost ? parseFloat(cost) : NaN;
  if (Number.isNaN(n)) return "—";
  return `$${n.toLocaleString("en-US", { minimumFractionDigits: digits, maximumFractionDigits: digits })}`;
}

const LIFECYCLE_TIPS: Record<string, string> = {
  Sleep: "Lifecycle: Sleep — the pod is deleted after each task, the workspace is kept",
  AutoDelete: "Lifecycle: AutoDelete — the agent is deleted after each task",
  Archive: "Lifecycle: Archive — pod and workspace are removed after each task, the record is kept",
};

type TtlSegment = { icon: typeof Moon; label: string; value: string; detail: string; next: boolean };

// The lifecycle timers shown as one segmented strip in the header. The sleep,
// archive or delete timer that fires soonest is marked `next`.
function buildTtlSegments(agent: AgentResponse): TtlSegment[] {
  const segs: TtlSegment[] = [];
  const timers: [string | undefined, string | undefined, typeof Moon, string, string][] = [
    [agent.sleepTTL, agent.sleepExpiresAt, Moon, "Sleep TTL", "Sleeps"],
    [agent.archiveTTL, agent.archiveExpiresAt, Archive, "Archive TTL", "Archives"],
    [agent.deleteTTL, agent.deleteExpiresAt, Hourglass, "Delete TTL", "Deletes"],
  ];
  let soonest = Infinity;
  for (const [ttl, at] of timers) {
    const t = ttl && at ? new Date(at).getTime() : NaN;
    if (!Number.isNaN(t) && t < soonest) soonest = t;
  }
  for (const [ttl, at, icon, label, verb] of timers) {
    if (!ttl) continue;
    const t = at ? new Date(at).getTime() : NaN;
    segs.push({ icon, label, value: fmtDuration(ttl), detail: ttlTitle(verb, at), next: t === soonest });
  }
  if (agent.taskTimeout) {
    segs.push({
      icon: Timer,
      label: "Task timeout",
      value: fmtDuration(agent.taskTimeout),
      detail: agent.taskExpiresAt ? ttlTitle("Cancels", agent.taskExpiresAt) : "Cancels a task that runs longer than this",
      next: false,
    });
  }
  return segs;
}

// A lifecycle-timer badge (sleep / delete TTL, task timeout) with an icon and a
// kit tooltip naming the timer and when it fires.
function TtlBadge({
  icon: Icon,
  label,
  detail,
  value,
}: {
  icon: typeof Moon;
  label: string;
  detail: string;
  value: string;
}) {
  return (
    <Tooltip
      side="bottom"
      content={
        <span className="flex flex-col gap-0.5">
          <span className="text-[var(--color-text)]">{label}</span>
          <span className="font-normal text-[var(--color-text-secondary)]">{detail}</span>
        </span>
      }
    >
      <Badge variant="secondary" className="text-xs gap-1.5 cursor-default" aria-label={`${label}: ${value}. ${detail}`}>
        <Icon className="size-3 text-[var(--color-text-muted)]" aria-hidden />
        {value}
      </Badge>
    </Tooltip>
  );
}

// Read-only list of why the agent is in its state (status.conditions). Hidden when empty.
function ConditionsSection({ conditions }: { conditions?: AgentCondition[] }) {
  if (!conditions || conditions.length === 0) {
    return <p className="text-[12.5px] text-[var(--color-text-secondary)]">No conditions reported yet.</p>;
  }
  return (
    <ul className="flex flex-col gap-2.5">
      {conditions.map((c) => {
        const active = c.status === "True";
        return (
          <li key={c.type} className="-mx-2 flex flex-col gap-0.5 rounded-[var(--radius-sm)] px-2 py-1 transition-colors hover:bg-[var(--color-surface-hover)]/60">
            <div className="flex items-center gap-2 text-[12.5px]">
              <span className="font-medium text-[var(--color-text)]">{c.type}</span>
              <Badge variant={active ? "default" : "outline"} className="text-[10px]">{c.status}</Badge>
              {c.lastTransitionTime && (
                <span className="ml-auto"><RelativeTime timestamp={c.lastTransitionTime} /></span>
              )}
            </div>
            {c.reason && <span className="text-[12px] text-[var(--color-text)]/85">{humanizeConditionReason(c.reason)}</span>}
            {c.message && <span className="text-[11.5px] text-[var(--color-text-secondary)]">{c.message}</span>}
          </li>
        );
      })}
    </ul>
  );
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
  const [teamUpOpen, setTeamUpOpen] = useState(false);
  const { squads, refresh: refreshSquads } = useSquads();

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

  // Archive now is asynchronous (the operator archives once the session is
  // saved), so poll until it shows up.
  const [archiving, setArchiving] = useState(false);
  const [archiveNote, setArchiveNote] = useState<string | null>(null);
  useEffect(() => {
    if (archiving && agent?.status === "Archived") {
      setArchiving(false);
      setArchiveNote(null);
    }
  }, [archiving, agent?.status]);
  useEffect(() => {
    if (!archiving) return;
    const timer = window.setTimeout(() => {
      setArchiving(false);
      setArchiveNote("Archive requested — it finishes once the session is saved");
    }, 180_000);
    return () => window.clearTimeout(timer);
  }, [archiving]);

  // Poll only while status is transient (Pending, Running with an active task, or archiving)
  const needsPolling = agent != null && (
    agent.status === "Pending" ||
    agent.taskStatus === "InProgress" ||
    agent.taskStatus === "Compacting" ||
    archiving
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

  // Find the squad this agent belongs to (if any)
  const agentSquad = agent
    ? squads.find((s) => s.members.some((m) => m.name === agent.name))
    : undefined;

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

  const [compactBusy, setCompactBusy] = useState(false);
  const [compactNote, setCompactNote] = useState<string | null>(null);
  // Set to the ISO time of the request when the user clicks Compact. The chat
  // shows an inline 'Compacting…' line until a compaction event with a timestamp
  // >= this value lands (the agent emitted one). Auto-clears after 30s to avoid
  // a stuck indicator if the agent never compacts (e.g. it stopped mid-request).
  const [compactionRequestedAt, setCompactionRequestedAt] = useState<string | null>(null);
  const handleCompact = async () => {
    if (!agentName || compactBusy) return;
    setCompactBusy(true);
    setCompactNote(null);
    try {
      await compactAgent(agentName, { namespace: agentNs });
      setCompactNote("Compacting…");
      setCompactionRequestedAt(new Date().toISOString());
      window.setTimeout(() => setCompactionRequestedAt(null), 30_000);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Compaction failed";
      // Friendly message for the most common case (no active task)
      setCompactNote(/409/.test(msg) || /no active task/i.test(msg) ? "No active task to compact" : msg);
    } finally {
      window.setTimeout(() => setCompactBusy(false), 2000);
      window.setTimeout(() => setCompactNote(null), 4000);
    }
  };

  // When a compaction event lands at or after the request timestamp, clear the
  // pending indicator — the divider will now appear in its place.
  // Hand off the indicator from the parent-prop (instant on click) to the
  // CR-derived signal (taskStatus === "Compacting") only once the CR has caught
  // up. That avoids a flicker between the two signals — the chat sees a
  // continuous indicator until the worker flips taskStatus back to InProgress.
  useEffect(() => {
    if (!compactionRequestedAt) return;
    if (agent?.taskStatus === "Compacting") {
      setCompactionRequestedAt(null);
    }
  }, [agent?.taskStatus, compactionRequestedAt]);

  const { s3Enabled } = usePlatformConfig();

  const handleDelete = async (opts?: { recreatePod?: boolean; force?: boolean }) => {
    if (!agentName) return;
    try {
      await deleteAgent(agentName, agentNs, opts);
      router.push("/agents");
    } catch {
      // Will be reflected in next poll
    }
  };

  const handleWake = async () => {
    if (!agentName) return;
    try {
      await wakeAgent(agentName, agentNs);
      fetchAgent();
    } catch {
      // Will be reflected in next poll
    }
  };

  const [sleepNote, setSleepNote] = useState<string | null>(null);
  const handleSleep = async () => {
    if (!agentName) return;
    setSleeping(true);
    setSleepNote(null);
    try {
      await sleepAgent(agentName, agentNs);
      await fetchAgent();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Failed to sleep agent";
      setSleepNote(/409/.test(msg) ? "Agent is running a task — cancel it first" : msg);
      window.setTimeout(() => setSleepNote(null), 4000);
    } finally {
      setSleeping(false);
    }
  };

  const handleArchive = async () => {
    if (!agentName) return;
    setArchiveNote(null);
    try {
      const res = await archiveAgent(agentName, agentNs);
      if (res.status !== "Archived") {
        setArchiving(true);
        setArchiveNote("Archiving…");
      }
      await fetchAgent();
    } catch (e: unknown) {
      setArchiveNote(e instanceof Error ? e.message : "Failed to archive agent");
      window.setTimeout(() => setArchiveNote(null), 4000);
    }
  };

  // Loading state
  if (showLoading) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex flex-1 items-center justify-center gap-2 text-sm text-[var(--color-text-secondary)]">
          <Loader2 className="size-4 animate-spin" />
          Loading agent…
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
        <div className="shrink-0 border-b border-[var(--color-border)] bg-[var(--color-bg)] pl-4 pr-2">
          <div className="@container flex flex-wrap items-center gap-x-3 gap-y-2 min-h-11 py-1.5">
            {/* Left: back button on settings view, otherwise status + model */}
            {view === "settings" ? (
              <Button variant="ghost" size="sm" onClick={() => setViewAndUrl("chat")}>
                <ArrowLeft className="size-3" data-icon="inline-start" />
                Back to chat
              </Button>
            ) : (
              <div className="flex min-w-0 flex-wrap items-center gap-2 whitespace-nowrap">
                <Tooltip
                  side="bottom"
                  content={`Agent ${agent.status}${agent.taskStatus ? ` · last task ${agent.taskStatus}` : ""}`}
                >
                  <span className="inline-flex items-center gap-2 rounded-full border border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 py-0.5 cursor-default">
                    <StatusBadge status={agent.status} />
                    {agent.taskStatus && (
                      <>
                        <span className="h-3 w-px bg-[var(--color-border-hover)]" aria-hidden />
                        <StatusBadge status={agent.taskStatus} size="sm" />
                      </>
                    )}
                  </span>
                </Tooltip>
                {agent.squad && agent.squadName && (
                  <Tooltip side="bottom" content={`Squad: ${agent.squadName}`}>
                    <Link
                      href={`/squads/${agent.squadName}?namespace=${agent.namespace}`}
                      className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] bg-[var(--color-brand-blue)]/10 text-[var(--color-brand-blue)] border border-[var(--color-brand-blue)]/20 hover:bg-[var(--color-brand-blue)]/20 transition-colors"
                    >
                      <Users className="size-3" />
                      <span>{agent.squadName}</span>
                    </Link>
                  </Tooltip>
                )}
                <Tooltip side="bottom" content={`Model: ${agent.model}`}>
                  <span className="block max-w-[140px] @7xl:max-w-[180px] truncate font-mono text-xs text-[var(--color-text-secondary)] cursor-default">
                    {agent.model}
                  </span>
                </Tooltip>
                {agent.effort && (
                  <Tooltip side="bottom" content={`Reasoning effort: ${agent.effort}`}>
                    <Badge variant="secondary" className="text-xs cursor-default">
                      {agent.effort}
                    </Badge>
                  </Tooltip>
                )}
                {agent.lifecycle && (
                  <Tooltip side="bottom" content={LIFECYCLE_TIPS[agent.lifecycle] ?? `Lifecycle: ${agent.lifecycle}`}>
                    <Badge variant="secondary" className="text-xs cursor-default">
                      {agent.lifecycle}
                    </Badge>
                  </Tooltip>
                )}
                {agent.storage?.ephemeral && (
                  <Tooltip
                    side="bottom"
                    content={
                      <span className="flex flex-col gap-0.5">
                        <span className="text-[var(--color-text)]">Ephemeral workspace</span>
                        <span className="font-normal text-[var(--color-text-secondary)]">
                          No PVC — files, packages, and the session are lost whenever the pod stops.
                        </span>
                      </span>
                    }
                  >
                    <Badge variant="secondary" className="text-xs gap-1.5 cursor-default" aria-label="Ephemeral workspace: no PVC, lost on pod restart">
                      <Eraser className="size-3 text-[var(--color-text-muted)]" aria-hidden />
                      Ephemeral
                    </Badge>
                  </Tooltip>
                )}
                {(() => {
                  const segs = buildTtlSegments(agent);
                  if (segs.length === 0) return null;
                  return (
                    <span className="inline-flex items-stretch rounded-full border border-[var(--color-border)] bg-[var(--color-surface)]" aria-label="Lifecycle timers">
                      {segs.map((t, i) => (
                        <Tooltip
                          key={t.label}
                          side="bottom"
                          content={
                            <span className="flex flex-col gap-0.5">
                              <span className="text-[var(--color-text)]">{t.label} · {t.value}{t.next ? " · next" : ""}</span>
                              <span className="font-normal text-[var(--color-text-secondary)]">{t.detail}</span>
                            </span>
                          }
                        >
                          <span
                            className={cn(
                              "inline-flex items-center gap-1 px-1.5 @7xl:px-2 py-0.5 text-xs tabular-nums cursor-default",
                              i > 0 && "border-l border-[var(--color-border)]",
                              t.next ? "text-[var(--color-status-sleeping)]" : "text-[var(--color-text-secondary)]",
                            )}
                            aria-label={`${t.label}: ${t.value}. ${t.detail}`}
                          >
                            <t.icon className="size-3" aria-hidden />
                            {t.value}
                          </span>
                        </Tooltip>
                      ))}
                    </span>
                  );
                })()}
                {agent.budgetUSD && (
                  <TtlBadge
                    icon={Wallet}
                    label="Budget"
                    detail="Soft lifetime spending cap across all tasks. New tasks are refused once reached — a running task still finishes."
                    value={`$${agent.totalCostUSD ?? "0"} / $${agent.budgetUSD}`}
                  />
                )}
                {agent.taskBudgetUSD && (
                  <TtlBadge
                    icon={Wallet}
                    label="Task budget"
                    detail="Hard per-task spending cap. The task stops the instant it is exceeded."
                    value={`$${agent.taskBudgetUSD}`}
                  />
                )}
                {agent.budgetExhausted && (
                  <Badge
                    variant="secondary"
                    className="text-xs gap-1.5 cursor-default border border-amber-500/40 bg-amber-500/10 text-amber-300"
                    aria-label="Budget exhausted: new tasks are refused"
                  >
                    <AlertTriangle className="size-3" aria-hidden />
                    Budget exhausted
                  </Badge>
                )}
              </div>
            )}

            {/* Right: usage, age, actions */}
            <div className="ml-auto flex flex-wrap items-center gap-x-3 gap-y-2 whitespace-nowrap [&_button]:whitespace-nowrap">
              <div className="flex items-baseline gap-2.5 text-xs tabular-nums">
                <Tooltip side="bottom" content={`Last task cost: ${fmtUSD(agent.lastTaskCostUSD, 4)}`}>
                  <span className="cursor-default">
                    <span className="font-mono font-medium text-[var(--color-text)]">{fmtUSD(agent.lastTaskCostUSD, 2)}</span>
                    <span className="ml-1 hidden @6xl:inline text-[var(--color-text-muted)]">last</span>
                  </span>
                </Tooltip>
                <Tooltip side="bottom" content={`Total cost: ${fmtUSD(agent.totalCostUSD, 4)}`}>
                  <span className="cursor-default">
                    <span className="font-mono font-medium text-[var(--color-text)]">{fmtUSD(agent.totalCostUSD, 2)}</span>
                    <span className="ml-1 hidden @6xl:inline text-[var(--color-text-muted)]">total</span>
                  </span>
                </Tooltip>
                {agent.totalTokens != null && agent.totalTokens > 0 && (
                  <Tooltip side="bottom" content={`Tokens used: ${agent.totalTokens.toLocaleString("en-US")}`}>
                    <span className="cursor-default">
                      <span className="font-mono font-medium text-[var(--color-text)]">{fmtTokens(agent.totalTokens)}</span>
                      <span className="ml-1 hidden @6xl:inline text-[var(--color-text-muted)]">tok</span>
                    </span>
                  </Tooltip>
                )}
              </div>

              <span className="inline-flex items-center gap-1 text-[var(--color-text-secondary)]">
                <CalendarDays className="size-3" aria-hidden />
                <RelativeTime timestamp={agent.createdAt} label="Created at" />
              </span>

              {agent.sessionStore && !agent.sessionStore.ok && (
                <Tooltip side="bottom" content={formatSessionStore(agent.sessionStore)}>
                  <span className="inline-flex text-[var(--color-status-error)]" aria-label={formatSessionStore(agent.sessionStore)}>
                    <CloudOff className="size-3.5" aria-hidden />
                  </span>
                </Tooltip>
              )}

              <div className="flex items-center gap-1.5">
                {/* One joined group for the agent's actions; deletes and settings sit outside it. */}
                <div className="inline-flex items-stretch rounded-full border border-[var(--color-border)] bg-[var(--color-surface)] [&>*+*]:border-l [&>*+*]:border-[var(--color-border)] [&_button]:!rounded-none [&_button]:!border-0 [&_button]:!bg-transparent [&_button]:!shadow-none [&_button]:!backdrop-blur-none [&_button:hover:not(:disabled)]:!bg-[var(--color-surface-hover)] [&>*:first-child_button]:!rounded-l-full [&>*:last-child_button]:!rounded-r-full">
                {archivedReadOnly(agent.status, s3Enabled) ? null : agentDormant(agent.status, s3Enabled) ? (
                  <Tooltip side="bottom" content="Wake the agent">
                    <Button variant="secondary" size="sm" onClick={handleWake} aria-label="Wake">
                      <Zap className="size-3" data-icon="inline-start" />
                      <span className="hidden @6xl:inline">Wake</span>
                    </Button>
                  </Tooltip>
                ) : (
                  <>
                    {(agent.taskStatus === "InProgress" || agent.taskStatus === "Compacting") && (
                      <Tooltip side="bottom" content="Cancel the running task">
                        <Button variant="secondary" size="sm" onClick={handleCancel} aria-label="Cancel task">
                          <Ban className="size-3" data-icon="inline-start" />
                          <span className="hidden @6xl:inline">Cancel</span>
                        </Button>
                      </Tooltip>
                    )}
                    <Tooltip
                      side="bottom"
                      content={
                        ephemeralBlocksSleep(agent.storage?.ephemeral, s3Enabled)
                          ? "Sleep is unavailable — this agent's workspace is ephemeral (no PVC). Sleeping deletes the pod, which would wipe it."
                          : "Sleep: delete the pod, keep the workspace"
                      }
                    >
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={handleSleep}
                        disabled={sleeping || ephemeralBlocksSleep(agent.storage?.ephemeral, s3Enabled)}
                        aria-label="Sleep"
                      >
                        <Moon className={`size-3 ${sleeping ? "animate-spin" : ""}`} data-icon="inline-start" />
                        <span className={sleepNote || sleeping ? undefined : "hidden @6xl:inline"}>
                          {sleepNote ?? (sleeping ? "Sleeping…" : "Sleep")}
                        </span>
                      </Button>
                    </Tooltip>
                  </>
                )}
                {agentSquad ? (
                  <LeaveSquadButton
                    agentName={agent.name}
                    agentNamespace={agent.namespace}
                    squad={agentSquad}
                    onSuccess={() => {
                      refreshSquads();
                      fetchAgent();
                    }}
                  />
                ) : !archivedReadOnly(agent.status, s3Enabled) && (
                  <Tooltip side="bottom" content="Team up with other agents in a squad">
                    <Button variant="secondary" size="sm" onClick={() => setTeamUpOpen(true)} aria-label="Team up">
                      <Users className="size-3" data-icon="inline-start" />
                      <span className="hidden @6xl:inline">Team Up</span>
                    </Button>
                  </Tooltip>
                )}
                {agent.status !== "Archived" && (
                <Tooltip
                  side="bottom"
                  content={
                    agent.status === "Sleeping"
                      ? "Wake the agent before compacting — there is no running pod to receive the request."
                      : "Compact the conversation history to free context. Runs as soon as the current turn finishes."
                  }
                >
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={handleCompact}
                    disabled={compactBusy || agent.status === "Sleeping"}
                    aria-label="Compact"
                    className="border-amber-500/40 bg-amber-500/10 text-amber-300 hover:bg-amber-500/15 hover:border-amber-500/55 hover:text-amber-200"
                  >
                    <Layers className="size-3" data-icon="inline-start" />
                    <span className={compactNote || compactBusy ? undefined : "hidden @6xl:inline"}>
                      {compactNote ?? (compactBusy ? "Compacting…" : "Compact")}
                    </span>
                  </Button>
                </Tooltip>
                )}
                {!agentSquad && s3Enabled && agent.status !== "Archived" && (
                    agent.taskStatus === "InProgress" || agent.taskStatus === "Compacting" || archiving ? (
                      // No confirm dialog behind a disabled button (a click would still
                      // reach the dialog's wrapper); the span carries the tooltip.
                      <span title={archiving ? undefined : "Archive is available once the current task finishes (or cancel it)."}>
                        <Button
                          variant="secondary"
                          size="sm"
                          disabled
                          className="border-[var(--color-status-archived)]/40 bg-[var(--color-status-archived)]/10 text-[var(--color-status-archived)]"
                        >
                          <Archive className="size-3" data-icon="inline-start" />
                          <span className={archiveNote ? undefined : "hidden @6xl:inline"}>{archiveNote ?? "Archive"}</span>
                        </Button>
                      </span>
                    ) : (
                      <Tooltip side="bottom" content="Archive: keep the record, remove the pod and workspace">
                      <ConfirmDialog
                        title="Archive Agent"
                        description={`Archive "${agent.name}"? Its pod and workspace files are removed once it's idle. The agent, its settings, cost and conversation are kept, and you can send it a task again later.`}
                        onConfirm={handleArchive}
                        confirmLabel="Archive"
                        confirmVariant="primary"
                        confirmClassName="bg-[var(--color-status-archived)] text-white hover:bg-[var(--color-status-archived)]/90 border-transparent"
                        trigger={
                          <Button
                            variant="secondary"
                            size="sm"
                            className="border-[var(--color-status-archived)]/40 bg-[var(--color-status-archived)]/10 text-[var(--color-status-archived)] hover:bg-[var(--color-status-archived)]/15 hover:border-[var(--color-status-archived)]/55"
                          >
                            <Archive className="size-3" data-icon="inline-start" />
                            <span className={archiveNote ? undefined : "hidden @6xl:inline"}>{archiveNote ?? "Archive"}</span>
                          </Button>
                        }
                      />
                      </Tooltip>
                    )
                )}
                </div>
                {s3Enabled && agent.sessionStore?.ok === false && (
                  <ConfirmDialog
                    title="Delete without saving"
                    description={`The last attempt to save "${agent.name}"'s session to S3 failed (${agent.sessionStore.error ?? "unknown error"}). A normal delete waits until the session is saved; this deletes the agent right away and its latest session is lost.`}
                    onConfirm={() => handleDelete({ force: true })}
                    trigger={
                      <Button variant="destructive" size="sm">
                        <Trash2 className="size-3" data-icon="inline-start" />
                        Delete without saving
                      </Button>
                    }
                  />
                )}
                {agentSquad ? (
                  <SquadAwareDeleteButton
                    agentName={agent.name}
                    agentNamespace={agent.namespace}
                    squad={agentSquad}
                    onConfirm={handleDelete}
                  />
                ) : s3Enabled ? (
                  // With the S3 session store, archiving keeps everything and the
                  // agent can run again, so it's how a live agent is retired;
                  // Delete is only offered once it's archived.
                  agent.status === "Archived" && (
                    <ConfirmDialog
                      title="Delete Agent"
                      description={`Are you sure you want to delete "${agent.name}"? This action cannot be undone. Its last saved session stays in S3.`}
                      onConfirm={handleDelete}
                      trigger={
                        <Button variant="destructive" size="sm" aria-label="Delete" title="Delete">
                          <Trash2 className="size-3" data-icon="inline-start" />
                          <span className="hidden @6xl:inline">Delete</span>
                        </Button>
                      }
                    />
                  )
                ) : (
                  <ConfirmDialog
                    title="Delete Agent"
                    description={`Are you sure you want to delete "${agent.name}"? This action cannot be undone.`}
                    onConfirm={handleDelete}
                    trigger={
                      <Button variant="destructive" size="sm" aria-label="Delete" title="Delete">
                        <Trash2 className="size-3" data-icon="inline-start" />
                        <span className="hidden @6xl:inline">Delete</span>
                      </Button>
                    }
                  />
                )}
                <Tooltip
                  content={view === "settings" ? "Back to chat" : "Settings"}
                  side="bottom"
                >
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setViewAndUrl(view === "settings" ? "chat" : "settings")}
                    aria-label={view === "settings" ? "Back to chat" : "Settings"}
                    className="!px-0 w-7 !bg-[var(--color-text)] !text-[var(--color-bg)] hover:!bg-[var(--color-text)]/90 border-[var(--color-text)]/20"
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
            {(() => {
              const primaryChat = (
                <AgentChat
                  agentName={agent.name}
                  agentNamespace={agentNs}
                  agentStatus={agent.status}
                  agentLifecycle={agent.lifecycle}
                  agentContextWindow={agent.modelContextWindow}
                  agentModel={agent.model}
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
                  compactionPending={compactionRequestedAt != null}
                />
              );
              if (agentSquad && agentSquad.members.length > 1) {
                return (
                  <SquadChatView
                    squadName={agentSquad.name}
                    members={agentSquad.members.map((m) => ({ name: m.name, namespace: agentNs || "default" }))}
                    primaryAgentName={agent.name}
                    primaryChat={primaryChat}
                  />
                );
              }
              return primaryChat;
            })()}
            <SubAgentPanel agentName={agent.name} events={events} namespace={agentNs} />
          </div>
        ) : (
          <div className="@container flex-1 overflow-y-auto">
            <div className="grid min-h-full grid-cols-[minmax(0,1fr)_340px] @max-5xl:grid-cols-1">
              {/* Form column: full-bleed section rows */}
              <div className="flex min-w-0 flex-col">
                {archivedReadOnly(agent.status, s3Enabled) ? (
                  <div className="border-b border-[var(--color-border)] px-7 py-4 text-sm text-[var(--color-text-secondary)]">
                    This agent is archived. Its pod and workspace were removed; its settings and cost are kept as a record.
                  </div>
                ) : (
                  <SettingsCard agent={agent} agentNs={agentNs} onSaved={(updated) => { if (updated) setAgent(updated); fetchAgent(); }} />
                )}
              </div>

              {/* Rail: overview, conditions, topology — a flush column, no cards */}
              <aside className="min-w-0 border-l border-[var(--color-border)] bg-[var(--color-bg-subtle)] @max-5xl:border-l-0 @max-5xl:border-t">
                <RailSection title="Overview">
                  <InfoRow label="Name" value={agent.name} mono />
                  <InfoRow label="Namespace" value={agent.namespace} mono />
                  <InfoRow label="Created" value={new Date(agent.createdAt).toLocaleString()} tip={`Created ${formatRelativeTime(agent.createdAt)}`} />
                  {agent.sessionStore && (
                    <InfoRow
                      label="Session"
                      tip={agent.sessionStore.ok ? "The latest conversation is saved to the S3 session store" : agent.sessionStore.error || "The last save failed"}
                      value={formatSessionStore(agent.sessionStore)}
                      color={agent.sessionStore.ok ? "var(--color-status-running)" : "var(--color-status-error)"}
                    />
                  )}
                  <div className="-mx-2 flex items-center justify-between gap-4 rounded-[var(--radius-sm)] px-2 py-0.5 transition-colors hover:bg-[var(--color-surface-hover)]/60">
                    <span className="text-[12.5px] font-medium text-[var(--color-text)] shrink-0">Squad</span>
                    {agentSquad ? (
                      <Link
                        href={`/squads/${agentSquad.name}?namespace=${agentSquad.namespace}`}
                        className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[12px] bg-[var(--color-brand-blue)]/10 text-[var(--color-brand-blue)] border border-[var(--color-brand-blue)]/20 hover:bg-[var(--color-brand-blue)]/20 transition-colors"
                      >
                        <Users className="size-3 shrink-0" />
                        {agentSquad.name}
                      </Link>
                    ) : (
                      <span className="text-[13px] text-[var(--color-text-secondary)]">Not in a squad</span>
                    )}
                  </div>
                  {archivedReadOnly(agent.status, s3Enabled) && agent.budgetUSD && (
                    <BudgetMeter spentUSD={agent.totalCostUSD} budgetUSD={agent.budgetUSD} exhausted={agent.budgetExhausted} />
                  )}
                </RailSection>
                <RailSection title="Conditions">
                  <ConditionsSection conditions={agent.conditions} />
                </RailSection>
                <AgentTopology agentName={agent.name} agentNs={agentNs} flush />
              </aside>
            </div>
          </div>
        )}
      </div>

      <TeamUpDialog
        open={teamUpOpen}
        onOpenChange={setTeamUpOpen}
        agentName={agent.name}
        agentNamespace={agent.namespace || agentNs || "default"}
        agentStatus={agent.status}
        squads={squads}
        onSuccess={refreshSquads}
      />
    </motion.div>
  );
}



// Spend-vs-budget meter: a small horizontal bar under the metrics row, shown
// only when a soft lifetime budgetUSD is set. Turns the accent color red once
// budgetExhausted (computed by the API) so the "new tasks refused" state is
// visible at a glance, matching the TTL badges' icon + detail convention.
function BudgetMeter({ spentUSD, budgetUSD, exhausted }: {
  spentUSD?: string;
  budgetUSD: string;
  exhausted?: boolean;
}) {
  const spent = parseFloat(spentUSD || "0") || 0;
  const budget = parseFloat(budgetUSD) || 0;
  const pct = budget > 0 ? Math.min(100, (spent / budget) * 100) : 0;
  return (
    <div className="flex flex-col gap-1">
      <div className="h-1.5 w-full rounded-full bg-[var(--color-surface-raised)] overflow-hidden" aria-label={`Spent $${spent.toFixed(2)} of $${budget.toFixed(2)}`}>
        <div
          className="h-full rounded-full transition-[width] duration-300"
          style={{ width: `${pct}%`, backgroundColor: exhausted ? "var(--color-status-error)" : "var(--color-brand-blue)" }}
        />
      </div>
      <span className={cn("text-[11.5px]", exhausted ? "text-[var(--color-status-error)]" : "text-[var(--color-text-muted)]")}>
        {exhausted ? "Budget exhausted — new tasks are refused until it's raised." : `$${spent.toFixed(2)} of $${budget.toFixed(2)} spent · new tasks stop at the cap`}
      </span>
    </div>
  );
}

function SettingsCard({ agent, agentNs, onSaved }: {
  agent: AgentResponse;
  agentNs?: string;
  onSaved: (updated?: AgentResponse) => void;
}) {
  const [model, setModel] = useState(agent.model);
  const [effort, setEffort] = useState<string>(agent.effort || "auto");
  const [lifecycle, setLifecycle] = useState<string>(agent.lifecycle || "default");
  const { s3Enabled } = usePlatformConfig();
  const sleepBlocked = ephemeralBlocksSleep(agent.storage?.ephemeral, s3Enabled);
  // TTLs round-trip as Go duration strings; show the compact form so editing "30m"
  // doesn't turn into "30m0s" on every save.
  const [sleepTTL, setSleepTTL] = useState(fmtDuration(agent.sleepTTL ?? ""));
  const [archiveTTL, setArchiveTTL] = useState(fmtDuration(agent.archiveTTL ?? ""));
  const [deleteTTL, setDeleteTTL] = useState(fmtDuration(agent.deleteTTL ?? ""));
  const [taskTimeout, setTaskTimeout] = useState(fmtDuration(agent.taskTimeout ?? ""));
  // Decimal USD strings, not Go durations — no fmtDuration needed.
  const [budgetUSD, setBudgetUSD] = useState(agent.budgetUSD ?? "");
  const [taskBudgetUSD, setTaskBudgetUSD] = useState(agent.taskBudgetUSD ?? "");
  const instructions = agent.instructions ?? "";
  const [systemPrompt, setSystemPrompt] = useState(agent.systemPrompt ?? "");
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
  const [labelRows, setLabelRows] = useState<LabelRow[]>(() => splitLabels(agent.labels).user);
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
  const agentEffort = agent.effort || "auto";
  const memoriesChanged = JSON.stringify(agentMemories.sort()) !== JSON.stringify((agent.memories ?? []).sort());
  const skillsChanged = JSON.stringify(agentSkills.sort()) !== JSON.stringify((agent.skills ?? []).sort());
  const connectorsChanged = JSON.stringify(agentConnectors.sort()) !== JSON.stringify((agent.connectors ?? []).sort());
  const secretsChanged = JSON.stringify(agentSecretRefs.sort()) !== JSON.stringify((agent.secrets ?? []).sort());
  const systemPromptChanged = systemPrompt !== (agent.systemPrompt ?? "");
  const sleepTTLChanged = sleepTTL.trim() !== fmtDuration(agent.sleepTTL ?? "");
  const archiveTTLChanged = archiveTTL.trim() !== fmtDuration(agent.archiveTTL ?? "");
  const deleteTTLChanged = deleteTTL.trim() !== fmtDuration(agent.deleteTTL ?? "");
  const taskTimeoutChanged = taskTimeout.trim() !== fmtDuration(agent.taskTimeout ?? "");
  const budgetUSDChanged = budgetUSD.trim() !== (agent.budgetUSD ?? "");
  const taskBudgetUSDChanged = taskBudgetUSD.trim() !== (agent.taskBudgetUSD ?? "");
  const { user: _userLabels, system: systemLabels } = splitLabels(agent.labels);
  const originalUserLabels = Object.fromEntries(_userLabels.map((r) => [r.key, r.value]));
  const labelErrors = validateLabelRows(labelRows);
  const labelsInvalid = hasLabelErrors(labelErrors);
  const labelDiff = diffLabels(agent.labels, labelRows);
  const labelsChanged = Object.keys(labelDiff.labels).length > 0 || labelDiff.removeLabels.length > 0;

  // Field names for the save bar, in form order.
  const changedFields = ([
    [systemPromptChanged, "System prompt"],
    [model !== agent.model, "Model"],
    [effort !== agentEffort, "Reasoning effort"],
    [lifecycle !== agentLifecycle, "Lifecycle"],
    [sleepTTLChanged, "Sleep TTL"],
    [archiveTTLChanged, "Archive TTL"],
    [deleteTTLChanged, "Delete TTL"],
    [taskTimeoutChanged, "Task timeout"],
    [budgetUSDChanged, "Budget"],
    [taskBudgetUSDChanged, "Task budget"],
    [connectorsChanged, "Connectors"],
    [skillsChanged, "Skills"],
    [memoriesChanged, "Memories"],
    [secretsChanged, "Secrets"],
    [labelsChanged, "Labels"],
  ] as [boolean, string][]).filter(([c]) => c).map(([, n]) => n);

  function discard() {
    setModel(agent.model);
    setEffort(agentEffort);
    setLifecycle(agentLifecycle);
    setSleepTTL(fmtDuration(agent.sleepTTL ?? ""));
    setArchiveTTL(fmtDuration(agent.archiveTTL ?? ""));
    setDeleteTTL(fmtDuration(agent.deleteTTL ?? ""));
    setTaskTimeout(fmtDuration(agent.taskTimeout ?? ""));
    setBudgetUSD(agent.budgetUSD ?? "");
    setTaskBudgetUSD(agent.taskBudgetUSD ?? "");
    setSystemPrompt(agent.systemPrompt ?? "");
    setAgentSecretRefs(agent.secrets ?? []);
    setAgentMemories(agent.memories ?? []);
    setAgentSkills(agent.skills ?? []);
    setAgentConnectors(agent.connectors ?? []);
    setLabelRows(splitLabels(agent.labels).user);
    setError(null);
  }

  // Live countdowns under the TTL fields; the sleep/archive/delete timer that fires
  // soonest is highlighted, as in the header strip.
  const nextTimer = buildTtlSegments(agent).find((t) => t.next)?.label;
  const ttlHint = (orig: string | undefined, changed: boolean, label: string, at: string | undefined, verb: string, fallback: string) => {
    if (changed) return { hint: `Was ${orig ? fmtDuration(orig) : "unset"}`, tone: "changed" as const };
    const when = fmtIn(at);
    if (orig && when) return { hint: `${verb} ${when}`, tone: nextTimer === label ? ("next" as const) : undefined };
    return { hint: fallback, tone: undefined };
  };
  const sleepH = ttlHint(agent.sleepTTL, sleepTTLChanged, "Sleep TTL", agent.sleepExpiresAt, "sleeps", "Sleep after this long idle");
  const archiveH = ttlHint(agent.archiveTTL, archiveTTLChanged, "Archive TTL", agent.archiveExpiresAt, "archives", "Archive after this long idle");
  const deleteH = ttlHint(agent.deleteTTL, deleteTTLChanged, "Delete TTL", agent.deleteExpiresAt, "deletes", "Delete this long after creation");
  const taskH = ttlHint(agent.taskTimeout, taskTimeoutChanged, "Task timeout", agent.taskExpiresAt, "cancels", agent.taskTimeout ? "No task running" : "Cap on a single task");

  async function handleSave() {
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      const patch: Record<string, unknown> = {};
      if (model !== agent.model) patch.model = model;
      if (effort !== agentEffort) patch.effort = effort === "auto" ? "" : effort;
      if (lifecycle !== agentLifecycle) patch.lifecycle = lifecycle === "default" ? "" : lifecycle;
      if (secretsChanged) patch.secretRefs = agentSecretRefs;
      if (memoriesChanged) patch.memories = agentMemories;
      if (skillsChanged) patch.skills = agentSkills;
      if (connectorsChanged) patch.connectors = agentConnectors;
      if (systemPromptChanged) patch.systemPrompt = systemPrompt.trim() || "";
      // Send "" to clear a TTL — the API treats explicit empty as "remove" and an
      // omitted field as "leave unchanged".
      if (sleepTTLChanged) patch.sleepTTL = sleepTTL.trim();
      if (archiveTTLChanged) patch.archiveTTL = archiveTTL.trim();
      if (deleteTTLChanged) patch.deleteTTL = deleteTTL.trim();
      if (taskTimeoutChanged) patch.taskTimeout = taskTimeout.trim();
      if (budgetUSDChanged) patch.budgetUSD = budgetUSD.trim();
      if (taskBudgetUSDChanged) patch.taskBudgetUSD = taskBudgetUSD.trim();
      if (Object.keys(labelDiff.labels).length > 0) patch.labels = labelDiff.labels;
      if (labelDiff.removeLabels.length > 0) patch.removeLabels = labelDiff.removeLabels;
      const updated = await patchAgent(agent.name, patch, agentNs);
      if (updated.errors && updated.errors.length > 0) {
        // Saved successfully but some live-pod sync steps failed. Surface the error
        // so the user knows part of the change didn't fully apply.
        setError(`Saved, but: ${updated.errors.join("; ")}`);
      } else {
        setSaved(true);
        setTimeout(() => setSaved(false), 2000);
      }
      onSaved(updated);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="@container flex min-h-full flex-col">
      <DetailsSection index={0} icon={FileText} title="Prompt" dirty={systemPromptChanged}
        description="What the agent is told. Instructions come with each task; the system prompt is the persona it keeps every task.">
        <PromptEditor instructions={instructions} systemPrompt={systemPrompt} onSystemPromptChange={setSystemPrompt} />
      </DetailsSection>

      <DetailsSection index={1} icon={Cpu} title="Model" dirty={model !== agent.model || effort !== agentEffort}
        description="Applies when the next pod starts.">
        <div className="grid grid-cols-3 @max-3xl:grid-cols-1 gap-4">
          <div className="col-span-2 @max-3xl:col-span-1 min-w-0">
            <ModelSelector value={model} onChange={(v) => setModel(v)} />
          </div>
          <Field label="Reasoning effort" tip="How much the model thinks before answering. Auto uses the model's default.">
            <Select value={effort} onValueChange={(v) => v && setEffort(v)}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {EFFORTS.map((e) => (
                  <SelectItem key={e.value} value={e.value}>{e.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
        </div>
      </DetailsSection>

      <DetailsSection index={2} icon={Clock} title="Lifecycle & limits"
        dirty={lifecycle !== agentLifecycle || sleepTTLChanged || archiveTTLChanged || deleteTTLChanged || taskTimeoutChanged || budgetUSDChanged || taskBudgetUSDChanged}
        description="What happens when it's idle, and how much it may spend.">
        <div className="grid grid-cols-2 @max-3xl:grid-cols-1 gap-4">
          <Field label="Lifecycle" tip="What happens to the agent after each task: keep running, sleep, delete, or archive." hint={sleepBlocked ? "Sleep is unavailable — this agent's workspace is ephemeral (no PVC)." : LIFECYCLE_TIPS[lifecycle]?.split(" — ")[1] ?? "After each task the pod keeps running."}>
            <Select value={lifecycle} onValueChange={(v) => v && setLifecycle(v)}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {LIFECYCLES.filter((l) => !(sleepBlocked && l.value === "Sleep")).map((l) => (
                  <SelectItem key={l.value} value={l.value}>{l.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field label="Storage" tip="The agent's workspace volume. Set when the agent is created and can't change." hint="Fixed at creation.">
            <Input
              value={agent.storage?.ephemeral ? "Ephemeral (emptyDir, no PVC)" : `${agent.storage?.size || "Template default"} · PVC`}
              disabled
              className="opacity-70 cursor-not-allowed"
            />
          </Field>
        </div>
        <div className="grid grid-cols-4 @max-4xl:grid-cols-2 gap-4">
          <Field label="Sleep TTL" tip="Sleep after this long idle (pod deleted, workspace kept). Resets on any activity. Empty to never sleep." hint={sleepH.hint} hintTone={sleepH.tone}>
            <Input value={sleepTTL} onChange={(e) => setSleepTTL(e.target.value)} placeholder="e.g. 30m" />
          </Field>
          <Field label="Archive TTL" tip="Archive after this long idle (pod and workspace removed, record kept). Keeps counting while asleep. Empty to never archive." hint={archiveH.hint} hintTone={archiveH.tone}>
            <Input value={archiveTTL} onChange={(e) => setArchiveTTL(e.target.value)} placeholder="e.g. 24h" />
          </Field>
          <Field label="Delete TTL" tip="Delete the agent this long after it was created. Absolute: never resets. Empty to never delete." hint={deleteH.hint} hintTone={deleteH.tone}>
            <Input value={deleteTTL} onChange={(e) => setDeleteTTL(e.target.value)} placeholder="e.g. 720h" />
          </Field>
          <Field label="Task timeout" tip="Cancel a single task that runs longer than this. Steering doesn't extend it. Empty for no limit." hint={taskH.hint} hintTone={taskH.tone}>
            <Input value={taskTimeout} onChange={(e) => setTaskTimeout(e.target.value)} placeholder="e.g. 30m" />
          </Field>
        </div>
        <div className="grid grid-cols-2 @max-3xl:grid-cols-1 gap-4">
          <Field label="Budget (USD)" tip="Soft lifetime cap: new tasks are refused once total spend reaches it; a running task still finishes." hint={budgetUSDChanged ? `Was ${agent.budgetUSD ? `$${agent.budgetUSD}` : "unset"}` : agent.budgetUSD ? undefined : "Soft lifetime cap. Empty for no cap."} hintTone={budgetUSDChanged ? "changed" : undefined}>
            <Input value={budgetUSD} onChange={(e) => setBudgetUSD(e.target.value)} placeholder="e.g. 10.00" />
            {agent.budgetUSD && !budgetUSDChanged && (
              <BudgetMeter spentUSD={agent.totalCostUSD} budgetUSD={agent.budgetUSD} exhausted={agent.budgetExhausted} />
            )}
          </Field>
          <Field label="Task budget (USD)" tip="Hard cap per task: the task stops the moment it spends more than this." hint={taskBudgetUSDChanged ? `Was ${agent.taskBudgetUSD ? `$${agent.taskBudgetUSD}` : "unset"}` : "Hard stop for a single task. Empty for no cap."} hintTone={taskBudgetUSDChanged ? "changed" : undefined}>
            <Input value={taskBudgetUSD} onChange={(e) => setTaskBudgetUSD(e.target.value)} placeholder="e.g. 2.50" />
          </Field>
        </div>
      </DetailsSection>

      <DetailsSection index={3} icon={Plug} title="Tools & knowledge" dirty={connectorsChanged || skillsChanged || memoriesChanged || secretsChanged}
        description="The connectors, skills, memories and secrets it can reach.">
        <div className="grid grid-cols-2 @max-3xl:grid-cols-1 gap-4">
          <Field label="Connectors" tip="MCP servers the agent can call as tools.">
            <MultiSelect
              options={availableConnectors.map<MultiSelectOption>((c) => ({
                value: c.ref,
                label: c.name,
                secondary: c.ref.includes("/") ? c.namespace : null,
                searchTerms: [c.namespace, c.ref],
              }))}
              value={agentConnectors}
              onChange={setAgentConnectors}
              placeholder="Select connectors..."
              noun="connectors"
              searchPlaceholder="Search connectors..."
              emptyText="No connectors available"
            />
          </Field>
          <Field label="Tool permissions" tip="Which built-in and connector tools the agent may use.">
            {(agent.allowedTools?.length || agent.disallowedTools?.length) ? (
          <div className="flex flex-col gap-2 rounded border border-[var(--color-border)] p-2.5">
            {agent.allowedTools?.length ? (
              <div className="flex flex-col gap-1">
                <div className="flex items-center gap-1.5 text-xs text-[var(--color-text-secondary)]">
                  <AlertTriangle className="size-3 shrink-0" />
                  <span>Restricted to these tools only — all other built-in and connector tools are unavailable.</span>
                </div>
                <div className="flex flex-wrap gap-1">
                  {agent.allowedTools.map((t) => (
                    <Badge key={t} variant="secondary" className="font-mono">{t}</Badge>
                  ))}
                </div>
              </div>
            ) : null}
            {agent.disallowedTools?.length ? (
              <div className="flex flex-col gap-1">
                <span className="text-xs text-[var(--color-text-secondary)]">Blocked tools</span>
                <div className="flex flex-wrap gap-1">
                  {agent.disallowedTools.map((t) => (
                    <Badge key={t} variant="outline" className="font-mono line-through">{t}</Badge>
                  ))}
                </div>
              </div>
            ) : null}
          </div>
            ) : (
              <span className="text-[12.5px] text-[var(--color-text-secondary)]">All default tools · set restrictions when creating the agent</span>
            )}
          </Field>
          <Field label="Skills" tip="Reusable instructions the agent can load when relevant.">
            <MultiSelect
              options={availableSkills.map<MultiSelectOption>((s) => ({
                value: s.ref,
                label: s.name,
                secondary: s.ref.includes("/") ? s.namespace : null,
                searchTerms: [s.namespace, s.ref],
              }))}
              value={agentSkills}
              onChange={setAgentSkills}
              placeholder="Select skills..."
              noun="skills"
              searchPlaceholder="Search skills..."
              emptyText="No skills available"
            />
          </Field>
          <Field label="Memories" tip="Knowledge added to the agent's prompt on every task.">
            <MultiSelect
              options={availableMemories.map<MultiSelectOption>((m) => ({
                value: m.ref,
                label: m.name,
                secondary: m.ref.includes("/") ? m.namespace : null,
                searchTerms: [m.namespace, m.ref],
              }))}
              value={agentMemories}
              onChange={setAgentMemories}
              placeholder="Select memories..."
              noun="memories"
              searchPlaceholder="Search memories..."
              emptyText="No memories available"
            />
          </Field>
          <Field label="Secrets" tip="Kubernetes secrets exposed to the agent as environment variables.">
          <MultiSelect
            options={availableSecrets.map<MultiSelectOption>((s) => ({
              value: s.name,
              label: s.name,
              secondary: showAllSecrets ? s.namespace : null,
              searchTerms: [s.namespace],
            }))}
            value={agentSecretRefs}
            onChange={setAgentSecretRefs}
            placeholder="Select secrets..."
            noun="secrets"
            searchPlaceholder="Search secrets..."
            emptyText="No secrets available"
            headerExtra={
              <button
                type="button"
                onClick={() => setShowAllSecrets((v) => !v)}
                className={`w-full text-xs px-2 py-1 rounded border transition-colors cursor-pointer ${
                  showAllSecrets
                    ? "border-amber-500/50 bg-amber-500/10 text-amber-400"
                    : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
                }`}
              >
                {showAllSecrets ? "Showing all namespaces" : "Show all namespaces"}
              </button>
            }
            footerExtra={
              <button
                type="button"
                onClick={() => setCreateSecretOpen(true)}
                className="flex w-full items-center justify-center gap-1 text-xs px-2 py-1 rounded border border-dashed border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)] hover:text-[var(--color-text)] transition-colors cursor-pointer"
              >
                <Plus className="size-3" />
                New Secret
              </button>
            }
          />
          <CreateSecretModal
            open={createSecretOpen}
            onOpenChange={setCreateSecretOpen}
            onCreated={() => {
              listSecrets(agentNs || undefined, showAllSecrets).then((res) => {
                setAvailableSecrets((res.secrets ?? []).map((s) => ({ name: s.name, namespace: s.namespace })));
              }).catch(() => {});
            }}
          />
          </Field>
        </div>
      </DetailsSection>

      <DetailsSection index={4} icon={Tag} title="Labels" dirty={labelsChanged} description="For filtering and selectors.">
        <LabelsEditor
          rows={labelRows}
          errors={labelErrors}
          system={systemLabels}
          original={originalUserLabels}
          onChange={setLabelRows}
        />
      </DetailsSection>

      <SaveBar
        changed={changedFields}
        saving={saving}
        saved={saved}
        error={error}
        disabled={labelsInvalid}
        onDiscard={discard}
        onSave={handleSave}
      />
    </div>
  );
}

function SquadAwareDeleteButton({
  agentName,
  squad,
  onConfirm,
}: {
  agentName: string;
  agentNamespace: string;
  squad: Squad;
  onConfirm: (opts?: { recreatePod?: boolean }) => void;
}) {
  return (
    <SquadAwareDeleteDialog
      title={`Delete ${agentName}?`}
      squad={{ name: squad.name, namespace: squad.namespace }}
      onConfirm={async ({ recreatePod, deleteSquads }) => {
        if (deleteSquads) {
          await deleteSquad(squad.name, squad.namespace);
        }
        onConfirm({ recreatePod });
      }}
      trigger={
        <Button variant="destructive" size="sm">
          <Trash2 className="size-3" data-icon="inline-start" />
          Delete
        </Button>
      }
    />
  );
}

function InfoRow({ label, value, mono, color, highlight, tip }: {
  label: string;
  value: string;
  mono?: boolean;
  color?: string;
  highlight?: boolean;
  /** Tooltip for the value; defaults to the value itself (it may be truncated). */
  tip?: string;
}) {
  return (
    <div className="-mx-2 flex items-center justify-between gap-4 rounded-[var(--radius-sm)] px-2 py-0.5 transition-colors hover:bg-[var(--color-surface-hover)]/60">
      <span className="text-[12.5px] font-medium text-[var(--color-text)] shrink-0">{label}</span>
      <span className="min-w-0">
        <Tooltip side="left" content={tip ?? value}>
          <span
            className={`block text-[13px] truncate text-right cursor-default ${
              mono ? "font-[family-name:var(--font-mono)] text-[12.5px]" : ""
            } ${highlight ? "font-semibold text-[var(--color-brand-blue)]" : "text-[var(--color-text)]"}`}
            style={color ? { color } : undefined}
          >
            {value}
          </span>
        </Tooltip>
      </span>
    </div>
  );
}
