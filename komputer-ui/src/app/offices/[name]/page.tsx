"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { motion } from "framer-motion";
import { Trash2 } from "lucide-react";

import { Button } from "@/components/kit/button";
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@/components/kit/select";
import { StatusBadge } from "@/components/shared/status-badge";
import { CostBadge } from "@/components/shared/cost-badge";
import { EventCard } from "@/components/shared/event-card";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { OfficeMembersGrid } from "@/components/offices/office-members";
import { getOffice, getOfficeEvents, deleteOffice, listAgents } from "@/lib/api";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import type { OfficeResponse, AgentEvent } from "@/lib/types";

export default function OfficeDetailPage() {
  const params = useParams<{ name: string }>();
  const name = params.name;
  const searchParams = useSearchParams();
  const namespace = searchParams.get("namespace") ?? undefined;
  const router = useRouter();

  const [office, setOffice] = useState<OfficeResponse | null>(null);
  const [events, setEvents] = useState<AgentEvent[]>([]);
  const [existingAgents, setExistingAgents] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const showLoading = useDelayedLoading(loading);
  const [notFound, setNotFound] = useState(false);
  const [eventAgentFilter, setEventAgentFilter] = useState("");
  const [eventLimit, setEventLimit] = useState("50");

  const agentNames = useMemo(
    () => [...new Set(events.map((e) => e.agentName).filter(Boolean))].sort(),
    [events]
  );

  const filteredEvents = useMemo(() => {
    let result = events;
    if (eventAgentFilter) {
      result = result.filter((e) => e.agentName === eventAgentFilter);
    }
    const limit = parseInt(eventLimit, 10) || 50;
    return result.slice(-limit);
  }, [events, eventAgentFilter, eventLimit]);

  const fetchData = useCallback(async () => {
    try {
      const [officeData, eventsData, agentsData] = await Promise.all([
        getOffice(name, namespace),
        getOfficeEvents(name, 50, namespace),
        listAgents(),
      ]);
      setOffice(officeData);
      // API returns { office, events } not a plain array — deduplicate and sort
      const eventsRaw = Array.isArray(eventsData) ? eventsData : (eventsData as { events?: AgentEvent[] })?.events ?? [];
      const seen = new Set<string>();
      const eventsArr = eventsRaw.filter((e) => {
        const key = `${e.timestamp}:${e.type}:${e.agentName}:${e.payload?.content ?? e.payload?.text ?? e.payload?.message ?? ""}`;
        if (seen.has(key)) return false;
        seen.add(key);
        return true;
      });
      eventsArr.sort((a, b) => a.timestamp.localeCompare(b.timestamp));
      setEvents(eventsArr);
      setExistingAgents(new Set((agentsData.agents || []).map((a) => a.name)));
      setNotFound(false);
    } catch (e: unknown) {
      if (e instanceof Error && e.message.includes("not found")) {
        setNotFound(true);
      }
    } finally {
      setLoading(false);
    }
  }, [name, namespace]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  async function handleDelete() {
    try {
      await deleteOffice(name, namespace);
      router.push("/offices");
    } catch {
      // non-critical
    }
  }

  if (showLoading) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex-1 overflow-y-auto p-6">
          <SkeletonTable />
        </div>
      </div>
    );
  }

  // Still loading but delay hasn't elapsed yet — render nothing to avoid flash
  if (loading) {
    return null;
  }

  if (notFound || !office) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex flex-1 flex-col items-center justify-center gap-4 text-center">
          <p className="text-lg font-medium text-[var(--color-text)]">
            Office not found
          </p>
          <p className="text-sm text-[var(--color-text-secondary)]">
            The office &quot;{name}&quot; does not exist or has been deleted.
          </p>
          <Link href="/offices">
            <Button variant="secondary" size="sm">
              Back to Offices
            </Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: "easeOut" }}
        className="flex-1 overflow-y-auto p-6 space-y-8"
      >
        {/* Office info bar */}
        <div className="flex flex-wrap items-center gap-4">
          <StatusBadge status={office.phase} />
          <div className="flex items-center gap-1.5 text-sm text-[var(--color-text-secondary)]">
            <span>Manager:</span>
            <Link
              href={`/agents/${office.manager}`}
              className="font-medium text-[var(--color-brand-blue)] hover:underline"
            >
              {office.manager}
            </Link>
          </div>
          <div className="flex items-center gap-1.5 text-sm text-[var(--color-text-secondary)]">
            <span>Total cost:</span>
            <CostBadge cost={office.totalCostUSD} />
          </div>
          <div className="ml-auto">
            <ConfirmDialog
              title={`Delete ${office.name}?`}
              description="This will permanently delete this office and all its member agents. This action cannot be undone."
              onConfirm={handleDelete}
              trigger={
                <Button variant="ghost" size="sm">
                  <Trash2 className="size-3.5 text-[var(--color-text-secondary)] hover:text-red-400" />
                  Delete
                </Button>
              }
            />
          </div>
        </div>

        {/* Members */}
        <motion.section
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, ease: "easeOut", delay: 0.1 }}
        >
          <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
            Members ({office.members.length})
          </h2>
          {office.members.length > 0 ? (
            <OfficeMembersGrid
              members={office.members}
              manager={office.manager}
              existingAgents={existingAgents}
            />
          ) : (
            <p className="text-sm text-[var(--color-text-secondary)]">
              No members yet.
            </p>
          )}
        </motion.section>

        {/* Events */}
        <motion.section
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, ease: "easeOut", delay: 0.25 }}
        >
          <div className="mb-3 flex flex-wrap items-center gap-3">
            <h2 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
              Events
            </h2>
            <div className="ml-auto flex items-center gap-2">
              <Select value={eventAgentFilter} onValueChange={setEventAgentFilter}>
                <SelectTrigger className="h-7 w-40 text-xs">
                  <SelectValue placeholder="All agents" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">All agents</SelectItem>
                  {agentNames.map((n) => (
                    <SelectItem key={n} value={n}>{n}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select value={eventLimit} onValueChange={setEventLimit}>
                <SelectTrigger className="h-7 w-20 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {["25", "50", "100", "200"].map((n) => (
                    <SelectItem key={n} value={n}>{n}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          {filteredEvents.length > 0 ? (
            <div className="space-y-2">
              {filteredEvents.map((event, i) => (
                <div key={`${event.timestamp}-${event.agentName}-${i}`}>
                  {event.agentName && (
                    <span className="mb-0.5 block text-[13px] font-medium text-[var(--color-text-secondary)]">
                      {event.agentName}
                    </span>
                  )}
                  <EventCard event={event} />
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-[var(--color-text-secondary)]">
              No events yet.
            </p>
          )}
        </motion.section>
      </motion.div>
    </div>
  );
}
