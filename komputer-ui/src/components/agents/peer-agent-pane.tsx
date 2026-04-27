"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { X } from "lucide-react";
import { AgentChat } from "@/components/agents/agent-chat";
import { useWebSocket } from "@/hooks/use-websocket";
import { getAgent, getAgentEvents } from "@/lib/api";
import type { AgentEvent, AgentResponse } from "@/lib/types";

export interface PeerAgentPaneProps {
  agentName: string;
  agentNamespace?: string;
  onRemove: () => void;
}

/**
 * Self-contained pane that fetches its own agent state, events, and websocket
 * subscription, then renders an AgentChat. Used by SquadChatView for the 2nd
 * and 3rd panes in multi-chat mode (the URL agent's pane uses the page's
 * primary AgentChat instance instead).
 */
export function PeerAgentPane({ agentName, agentNamespace, onRemove }: PeerAgentPaneProps) {
  const [agent, setAgent] = useState<AgentResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [historyEvents, setHistoryEvents] = useState<AgentEvent[]>([]);
  const [hasMoreEvents, setHasMoreEvents] = useState(true);
  const [loadingOlder, setLoadingOlder] = useState(false);

  const { events: wsEvents } = useWebSocket(agentName);

  const parseEventsResponse = useCallback((data: unknown): AgentEvent[] => {
    return Array.isArray(data) ? data : (data as { events?: AgentEvent[] })?.events ?? [];
  }, []);

  // Initial event history fetch
  useEffect(() => {
    let cancelled = false;
    getAgentEvents(agentName, 50, agentNamespace)
      .then((data) => {
        if (cancelled) return;
        const arr = parseEventsResponse(data);
        setHistoryEvents(arr);
        if (arr.length < 50) setHasMoreEvents(false);
      })
      .catch(() => {
        if (cancelled) return;
        setHistoryEvents([]);
        setHasMoreEvents(false);
      });
    return () => {
      cancelled = true;
    };
  }, [agentName, agentNamespace, parseEventsResponse]);

  // Older-events pagination
  const historyEventsRef = useRef(historyEvents);
  historyEventsRef.current = historyEvents;
  const loadingOlderRef = useRef(false);
  const hasMoreEventsRef = useRef(hasMoreEvents);
  hasMoreEventsRef.current = hasMoreEvents;
  const scrollContainerRef = useRef<HTMLElement | null>(null);
  const scrollSnapshotRef = useRef<number | null>(null);

  const loadOlderEvents = useCallback(async () => {
    if (loadingOlderRef.current || !hasMoreEventsRef.current) return;
    const oldest = historyEventsRef.current;
    const oldestTimestamp = oldest.length > 0 ? oldest[0].timestamp : undefined;
    if (!oldestTimestamp) return;
    loadingOlderRef.current = true;
    setLoadingOlder(true);
    try {
      const data = await getAgentEvents(agentName, 50, agentNamespace, oldestTimestamp);
      const older = parseEventsResponse(data);
      if (older.length === 0) {
        setHasMoreEvents(false);
      } else {
        if (scrollContainerRef.current) {
          scrollSnapshotRef.current = scrollContainerRef.current.scrollHeight;
        }
        setHistoryEvents((prev) => [...older, ...prev]);
        if (older.length < 50) setHasMoreEvents(false);
      }
    } catch {
      // swallow; user can scroll up again
    } finally {
      loadingOlderRef.current = false;
      setLoadingOlder(false);
    }
  }, [agentName, agentNamespace, parseEventsResponse]);

  // Merge history + WS events, dedup by timestamp+type, sorted by time
  const events = useMemo(() => {
    const seen = new Set<string>();
    return [...historyEvents, ...wsEvents]
      .filter((e) => {
        const normType = e.type === "task_started" ? "user_message" : e.type;
        const key = `${e.timestamp}:${normType}`;
        if (seen.has(key)) return false;
        seen.add(key);
        return true;
      })
      .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
  }, [historyEvents, wsEvents]);

  // Fetch agent info
  const fetchAgent = useCallback(async () => {
    try {
      const data = await getAgent(agentName, agentNamespace);
      setAgent(data);
      setError(null);
    } catch (e: unknown) {
      if (e instanceof Error && e.message.includes("not found")) {
        setError("Agent no longer exists");
      } else {
        setError(e instanceof Error ? e.message : "Failed to load agent");
      }
    }
  }, [agentName, agentNamespace]);

  useEffect(() => {
    fetchAgent();
  }, [fetchAgent]);

  // Refetch on websocket events that may indicate a state change
  useEffect(() => {
    if (wsEvents.length === 0) return;
    fetchAgent();
  }, [wsEvents.length, fetchAgent]);

  if (error) {
    return (
      <div className="flex flex-1 min-w-0 flex-col items-center justify-center gap-3 border-l border-[var(--color-border)] p-6 text-center">
        <p className="text-sm text-[var(--color-text-secondary)]">
          <span className="font-medium text-[var(--color-text)]">{agentName}</span> — {error}
        </p>
        <button
          type="button"
          onClick={onRemove}
          className="text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text)] underline transition-colors cursor-pointer"
        >
          Close pane
        </button>
      </div>
    );
  }

  if (!agent) {
    return (
      <div className="flex flex-1 min-w-0 flex-col items-center justify-center border-l border-[var(--color-border)]">
        <p className="text-sm text-[var(--color-text-muted)]">Loading {agentName}…</p>
      </div>
    );
  }

  return (
    <div className="relative flex flex-1 min-w-0 flex-col border-l border-[var(--color-border)]">
      <button
        type="button"
        onClick={onRemove}
        className="absolute top-2 right-2 z-10 flex h-6 w-6 items-center justify-center rounded text-[var(--color-text-muted)] hover:bg-[var(--color-surface-hover)] hover:text-[var(--color-text)] transition-colors cursor-pointer"
        aria-label={`Close ${agentName} pane`}
      >
        <X className="size-3.5" />
      </button>
      <AgentChat
        agentName={agent.name}
        agentNamespace={agentNamespace}
        agentStatus={agent.status}
        agentLifecycle={agent.lifecycle}
        agentContextWindow={agent.modelContextWindow}
        events={events}
        taskStatus={agent.taskStatus}
        hasMoreEvents={hasMoreEvents}
        loadingOlder={loadingOlder}
        onLoadOlder={loadOlderEvents}
        scrollContainerRef={scrollContainerRef}
        scrollSnapshotRef={scrollSnapshotRef}
      />
    </div>
  );
}
