"use client";

import { useState, useEffect, useMemo, useCallback, useRef } from "react";
import { createPortal } from "react-dom";
import { motion, AnimatePresence } from "framer-motion";
import Link from "next/link";
import { Bot, Zap, Moon, Clock, Skull, CheckCircle2, Users } from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { AgentEvent } from "@/lib/types";
import { useWebSocket } from "@/hooks/use-websocket";
import { getOffice, listAgents, getAgentEvents } from "@/lib/api";
import type { OfficeMemberResponse } from "@/lib/types";

type SubAgentInfo = {
  name: string;
  instructions: string;
};

const statusIcons: Record<string, { icon: typeof Bot; color: string }> = {
  Running: { icon: Zap, color: "#34D399" },
  Sleeping: { icon: Moon, color: "#FBBF24" },
  Pending: { icon: Clock, color: "#FBBF24" },
  Failed: { icon: Skull, color: "#F87171" },
  Succeeded: { icon: CheckCircle2, color: "#34D399" },
  Deleted: { icon: Skull, color: "#F87171" },
};

const defaultIcon = { icon: Bot, color: "#8899A6" };

function LastEventPreview({ msg, onHoverChange }: { msg: { type: string; text: string; fullText: string }; onHoverChange: (h: boolean) => void }) {
  const [expanded, setExpanded] = useState(false);
  const expandTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isTruncated = msg.fullText.length > msg.text.length;
  const expandRef = useRef<HTMLDivElement>(null);
  const [expandPos, setExpandPos] = useState<{ top?: number; bottom?: number; left: number }>({ left: 0 });

  const showExpand = () => {
    if (expandTimeout.current) clearTimeout(expandTimeout.current);
    setExpanded(true);
    onHoverChange(true);
  };
  const hideExpand = () => {
    expandTimeout.current = setTimeout(() => {
      setExpanded(false);
      onHoverChange(false);
    }, 150);
  };

  const handleEnter = (e: React.MouseEvent) => {
    if (!isTruncated) return;
    showExpand();
    // Find the parent tooltip (closest fixed positioned ancestor with w-80 = 320px)
    const parentTooltip = (e.currentTarget as HTMLElement).closest("[data-sub-tooltip-parent]") as HTMLElement | null;
    const parentRect = parentTooltip?.getBoundingClientRect();
    const subWidth = 448; // w-[28rem]
    // Center the sub-tooltip relative to the parent tooltip
    const parentCenterX = parentRect
      ? parentRect.left + parentRect.width / 2
      : (e.currentTarget as HTMLElement).getBoundingClientRect().left;
    const left = Math.max(8, Math.min(parentCenterX - subWidth / 2, window.innerWidth - subWidth - 8));

    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const spaceBelow = window.innerHeight - rect.bottom;
    const maxH = Math.min(400, window.innerHeight * 0.5);
    if (spaceBelow > maxH + 20) {
      setExpandPos({ top: rect.bottom + 4, left });
    } else {
      setExpandPos({ bottom: window.innerHeight - rect.top + 4, left });
    }
  };

  return (
    <div>
      <div
        className={`text-sm leading-relaxed prose-chat line-clamp-2 ${msg.type === "thinking" ? "italic text-[var(--color-text-muted)]" : "text-[var(--color-text-secondary)]"} ${isTruncated ? "cursor-pointer hover:text-[var(--color-text)]" : ""}`}
        onMouseEnter={handleEnter}
        onMouseLeave={hideExpand}
      >
        <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.text}</ReactMarkdown>
      </div>
      {typeof window !== "undefined" && createPortal(
        <AnimatePresence>
          {expanded && (
            <motion.div
              ref={expandRef}
              initial={{ opacity: 0, y: expandPos.top ? 4 : -4 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: expandPos.top ? 4 : -4 }}
              transition={{ duration: 0.12 }}
              className="fixed z-[10000] w-[28rem] max-h-[50vh] overflow-y-auto rounded-lg border border-[var(--color-border)] bg-[#0a0e14] p-4 shadow-[0_12px_40px_rgba(0,0,0,0.7)]"
              style={{
                top: expandPos.top,
                bottom: expandPos.bottom,
                left: Math.max(8, expandPos.left),
              }}
              onMouseEnter={showExpand}
              onMouseLeave={hideExpand}
            >
              <div className="text-sm leading-relaxed prose-chat text-[var(--color-text)]">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.fullText}</ReactMarkdown>
              </div>
            </motion.div>
          )}
        </AnimatePresence>,
        document.body
      )}
    </div>
  );
}

function SubAgentCard({ info, namespace, exists, agentPhase, memberStatus }: { info: SubAgentInfo; namespace?: string; exists: boolean; agentPhase?: string; memberStatus?: string }) {
  const { events: wsEvents } = useWebSocket(exists ? info.name : null);
  const [historyEvents, setHistoryEvents] = useState<AgentEvent[]>([]);
  const [hovered, setHovered] = useState(false);
  const hoverTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const [tooltipPos, setTooltipPos] = useState({ top: 0, right: 0 });

  const showTooltip = () => {
    if (hoverTimeout.current) clearTimeout(hoverTimeout.current);
    setHovered(true);
  };
  const hideTooltip = () => {
    hoverTimeout.current = setTimeout(() => setHovered(false), 150);
  };

  // Fetch last 5 events from history on mount
  useEffect(() => {
    if (!exists) return;
    getAgentEvents(info.name, 5, namespace)
      .then((data: unknown) => {
        const arr = Array.isArray(data) ? data : (data as { events?: AgentEvent[] })?.events ?? [];
        setHistoryEvents(arr);
      })
      .catch(() => {});
  }, [info.name, namespace, exists]);

  // Merge history + WS, dedup by timestamp+type
  const events = useMemo(() => {
    const all = [...historyEvents, ...wsEvents];
    const seen = new Set<string>();
    return all.filter((e) => {
      const key = `${e.timestamp}:${e.type}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    }).sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
  }, [historyEvents, wsEvents]);

  // Derive status: existence → API phase → live events → office member data
  const status = useMemo(() => {
    if (!exists) return "Deleted";
    // API phase is authoritative for Sleeping/Failed/Succeeded
    if (agentPhase === "Sleeping") return "Sleeping";
    if (agentPhase === "Failed") return "Failed";
    // Live events for real-time task status
    for (let i = events.length - 1; i >= 0; i--) {
      const t = events[i].type;
      if (t === "task_completed") return "Succeeded";
      if (t === "task_cancelled") return "Succeeded";
      if (t === "error") return "Failed";
      if (t === "task_started" || t === "thinking" || t === "tool_call" || t === "text") return "Running";
    }
    // Fall back to API phase
    if (agentPhase === "Running") return "Running";
    if (agentPhase === "Succeeded") return "Succeeded";
    // Fall back to office member status
    if (memberStatus === "Complete") return "Succeeded";
    if (memberStatus === "InProgress") return "Running";
    if (memberStatus === "Error") return "Failed";
    return "Pending";
  }, [events, exists, agentPhase, memberStatus]);

  // Last few messages for preview
  const preview = useMemo(() => {
    return events
      .filter((e) => e.type === "text" || e.type === "thinking")
      .slice(-3)
      .map((e) => ({
        type: e.type,
        text: (e.payload.content ?? e.payload.text ?? "").slice(0, 120),
        fullText: e.payload.content ?? e.payload.text ?? "",
      }));
  }, [events]);

  const { icon: StatusIcon, color } = statusIcons[status] ?? defaultIcon;
  const isActive = status === "Running";

  const handleMouseEnter = () => {
    showTooltip();
    if (cardRef.current) {
      const rect = cardRef.current.getBoundingClientRect();
      setTooltipPos({ top: rect.top, right: window.innerWidth - rect.left + 8 });
    }
  };

  return (
    <div ref={cardRef}>
      <Link href={`/agents/${info.name}${namespace ? `?namespace=${namespace}` : ""}`}>
        <motion.div
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.2 }}
          className={`rounded-lg border bg-[var(--color-surface)] p-3 cursor-pointer transition-all duration-150 hover:border-[var(--color-border-hover)] hover:shadow-[0_0_12px_rgba(63,133,217,0.08)] ${
            isActive
              ? "border-[#34D399]/40 animate-[activeGlow_2s_ease-in-out_infinite]"
              : "border-[var(--color-border)]"
          }`}
          onMouseEnter={handleMouseEnter}
          onMouseLeave={hideTooltip}
        >
          <div className="flex items-center gap-2">
            <div
              className="flex size-7 items-center justify-center rounded-md shrink-0"
              style={{ backgroundColor: `${color}15` }}
            >
              <StatusIcon className="size-3.5" style={{ color }} />
            </div>
            <span className="text-sm font-medium text-[var(--color-text)] truncate flex-1 min-w-0">
              {info.name}
            </span>
            <span
              className="size-2 shrink-0 rounded-full"
              style={{ backgroundColor: color, boxShadow: status === "Running" ? `0 0 6px ${color}80` : undefined }}
            />
          </div>
          <div className="mt-1.5 text-xs text-[var(--color-text-muted)] line-clamp-2 leading-relaxed prose-chat [&_p]:m-0">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{info.instructions.slice(0, 120)}</ReactMarkdown>
          </div>
        </motion.div>
      </Link>

      {/* Hover tooltip with status + preview — rendered via portal to escape overflow */}
      {typeof window !== "undefined" && createPortal(
        <AnimatePresence>
          {hovered && (
            <motion.div
              initial={{ opacity: 0, x: 6 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: 6 }}
              transition={{ duration: 0.15 }}
              data-sub-tooltip-parent
              className="fixed z-[9999] w-80 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-raised)] p-3 shadow-[0_8px_32px_rgba(0,0,0,0.4)]"
              style={{ top: tooltipPos.top, right: tooltipPos.right }}
              onMouseEnter={showTooltip}
              onMouseLeave={hideTooltip}
            >
            {/* Header */}
            <div className="flex items-center gap-2 pb-2 mb-2 border-b border-[var(--color-border)]">
              <StatusIcon className="size-3.5" style={{ color }} />
              <div className="min-w-0 flex-1">
                <p className="text-xs font-semibold text-[var(--color-text)] truncate">{info.name}</p>
                <p className="text-[10px] text-[var(--color-text-muted)]">{namespace || "default"}</p>
              </div>
            </div>

            {/* Details */}
            <div className="space-y-1.5 mb-2">
              <div className="flex justify-between gap-4">
                <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">Status</span>
                <span className="text-[11px] font-medium" style={{ color }}>{status}</span>
              </div>
              {memberStatus && (
                <div className="flex justify-between gap-4">
                  <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">Task</span>
                  <span className="text-[11px] font-medium text-[var(--color-text)]">{memberStatus}</span>
                </div>
              )}
            </div>

            {/* Last event */}
            {preview.length > 0 ? (
              <div className="pt-2 border-t border-[var(--color-border)]">
                <p className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)] mb-1.5">Latest</p>
                <LastEventPreview msg={preview[preview.length - 1]} onHoverChange={(h) => { if (h) showTooltip(); else hideTooltip(); }} />
              </div>
            ) : (
              <p className="text-[10px] text-[var(--color-text-muted)] pt-2 border-t border-[var(--color-border)]">Waiting for activity...</p>
            )}
          </motion.div>
        )}
      </AnimatePresence>,
      document.body
      )}
    </div>
  );
}

export function SubAgentPanel({ agentName, events, namespace }: { agentName: string; events: AgentEvent[]; namespace?: string }) {
  // Office members (fetched on mount)
  const [officeMembers, setOfficeMembers] = useState<{ name: string; role: string }[]>([]);
  const [existingAgents, setExistingAgents] = useState<Set<string> | null>(null);
  const [agentStatuses, setAgentStatuses] = useState<Record<string, string>>({});
  const [memberStatuses, setMemberStatuses] = useState<Record<string, string>>({});

  const fetchStatuses = useCallback(() => {
    const officeName = `${agentName}-office`;
    getOffice(officeName, namespace)
      .then((office) => {
        const members = (office.members || []).filter((m) => m.name !== agentName);
        setOfficeMembers(members.map((m) => ({ name: m.name, role: m.role })));
        const statuses: Record<string, string> = {};
        for (const m of members) {
          if (m.taskStatus) statuses[m.name] = m.taskStatus;
        }
        setMemberStatuses(statuses);
      })
      .catch(() => {});
    listAgents(namespace)
      .then((res) => {
        const agents = res.agents || [];
        setExistingAgents(new Set(agents.map((a) => a.name)));
        const statuses: Record<string, string> = {};
        for (const a of agents) {
          statuses[a.name] = a.status;
        }
        setAgentStatuses(statuses);
      })
      .catch(() => setExistingAgents(new Set()));
  }, [agentName, namespace]);

  // Initial fetch
  useEffect(() => { fetchStatuses(); }, [fetchStatuses]);

  // Extract sub-agents from events (create_agent tool calls)
  const eventAgents = useMemo(() => {
    const agents: SubAgentInfo[] = [];
    const seen = new Set<string>();
    for (const event of events) {
      if (event.type === "tool_call" && (event.payload.tool === "create_agent" || event.payload.tool === "mcp__komputer__create_agent")) {
        const name = event.payload.input?.name;
        const instructions = event.payload.input?.instructions ?? "";
        if (name && !seen.has(name)) {
          seen.add(name);
          agents.push({ name, instructions });
        }
      }
    }
    return agents;
  }, [events]);

  // Merge: event-detected agents + office members, dedup by name
  const subAgents = useMemo(() => {
    const seen = new Set<string>();
    const result: SubAgentInfo[] = [];
    // Event agents first (have instructions)
    for (const a of eventAgents) {
      seen.add(a.name);
      result.push(a);
    }
    // Office members that weren't in events
    for (const m of officeMembers) {
      if (!seen.has(m.name)) {
        seen.add(m.name);
        result.push({ name: m.name, instructions: "" });
      }
    }
    const isActive = (name: string) => {
      const phase = agentStatuses[name];
      const task = memberStatuses[name];
      return phase === "Running" || phase === "Pending" || task === "InProgress";
    };
    const isDead = (name: string) => {
      const phase = agentStatuses[name];
      return phase === "Failed" || phase === "Succeeded" || (existingAgents !== null && !existingAgents.has(name));
    };
    return result.sort((a, b) => {
      const aActive = isActive(a.name);
      const bActive = isActive(b.name);
      if (aActive !== bActive) return aActive ? -1 : 1;
      const aDead = isDead(a.name);
      const bDead = isDead(b.name);
      if (aDead !== bDead) return aDead ? 1 : -1;
      return a.name.localeCompare(b.name);
    });
  }, [eventAgents, officeMembers, agentStatuses, memberStatuses, existingAgents]);

  // Poll every 5s only when there are sub-agents
  useEffect(() => {
    if (subAgents.length === 0) return;
    const interval = setInterval(fetchStatuses, 5000);
    return () => clearInterval(interval);
  }, [subAgents.length, fetchStatuses]);

  // Track which agents were present on first render — skip animations for those (page refresh case).
  const initialAgentNames = useRef<Set<string> | null>(null);
  if (initialAgentNames.current === null) {
    initialAgentNames.current = new Set(subAgents.map((a) => a.name));
  }

  if (subAgents.length === 0) return null;

  const skipPanelAnimation = initialAgentNames.current.size > 0;

  return (
    <motion.div
      initial={skipPanelAnimation ? { opacity: 1, width: 260 } : { opacity: 0, width: 0 }}
      animate={{ opacity: 1, width: 260 }}
      transition={{ duration: 0.3, ease: "easeOut" }}
      className="shrink-0 border-l border-[var(--color-border)] bg-[var(--color-bg-subtle)] overflow-y-auto"
    >
      <div className="p-3 space-y-2">
        <div className="flex items-center gap-2 mb-3">
          <Users className="size-3.5 text-[var(--color-text-muted)]" />
          <span className="text-[11px] font-semibold uppercase tracking-wider text-[var(--color-text-muted)]">
            Sub-agents ({subAgents.length})
          </span>
        </div>
        {subAgents.map((agent, i) => (
          <motion.div
            key={agent.name}
            initial={initialAgentNames.current?.has(agent.name) ? { opacity: 1, y: 0 } : { opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2, delay: i * 0.05 }}
          >
            <SubAgentCard
              info={agent}
              namespace={namespace}
              exists={existingAgents === null ? true : existingAgents.has(agent.name)}
              agentPhase={agentStatuses[agent.name]}
              memberStatus={memberStatuses[agent.name]}
            />
          </motion.div>
        ))}
      </div>
    </motion.div>
  );
}
