"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { motion } from "framer-motion";
import { Trash2, Zap, Moon, AlertTriangle, CheckCircle2, Circle } from "lucide-react";

import { Button } from "@/components/kit/button";
import { StatusBadge } from "@/components/shared/status-badge";
import { CostBadge } from "@/components/shared/cost-badge";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { BreakUpButton } from "@/components/squads/break-up-button";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { SquadTopologyGraph } from "@/components/topology/topology-graph";
import {
  getSquad,
  deleteSquad,
  listAgents,
  patchAgent,
  createAgent,
} from "@/lib/api";
import type { Squad, AgentResponse } from "@/lib/types";

// ---- Orphan TTL countdown helpers ----

function parseDuration(ttl: string): number {
  // Supports formats: 30m, 2h, 1h30m, 90s
  let total = 0;
  const matches = ttl.matchAll(/(\d+)([smhd])/g);
  for (const m of matches) {
    const val = parseInt(m[1], 10);
    switch (m[2]) {
      case "s": total += val * 1000; break;
      case "m": total += val * 60 * 1000; break;
      case "h": total += val * 60 * 60 * 1000; break;
      case "d": total += val * 24 * 60 * 60 * 1000; break;
    }
  }
  return total;
}

function fmtCountdown(ms: number): string {
  if (ms <= 0) return "imminent";
  const s = Math.floor(ms / 1000);
  const m = Math.floor(s / 60);
  const h = Math.floor(m / 60);
  const d = Math.floor(h / 24);
  if (d > 0) return `${d}d ${h % 24}h`;
  if (h > 0) return `${h}h ${m % 60}m`;
  if (m > 0) return `${m}m ${s % 60}s`;
  return `${s}s`;
}

function useOrphanCountdown(orphanedSince?: string, orphanTTL?: string): string | null {
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    if (!orphanedSince || !orphanTTL) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [orphanedSince, orphanTTL]);

  if (!orphanedSince || !orphanTTL) return null;
  const since = new Date(orphanedSince).getTime();
  const ttlMs = parseDuration(orphanTTL);
  const deleteAt = since + ttlMs;
  return fmtCountdown(deleteAt - now);
}

// ---- Member card ----

function MemberCard({
  member,
  agent,
  namespace,
}: {
  member: { name: string; ready: boolean; taskStatus?: string };
  agent?: AgentResponse;
  namespace: string;
}) {
  return (
    <Link
      href={`/agents/${member.name}?namespace=${namespace}`}
      className="flex items-center gap-3 rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-3 transition-colors hover:border-[var(--color-border-hover)] hover:bg-[var(--color-surface-hover)]"
    >
      {/* Ready indicator */}
      <span className="shrink-0">
        {member.ready ? (
          <CheckCircle2 className="size-3.5 text-[#34D399]" />
        ) : (
          <Circle className="size-3.5 text-[var(--color-text-muted)]" />
        )}
      </span>

      {/* Name */}
      <span className="flex-1 truncate text-sm font-medium text-[var(--color-text)]">
        {member.name}
      </span>

      {/* Agent status badge */}
      {agent?.status && (
        <StatusBadge status={agent.status} size="sm" />
      )}

      {/* Task status */}
      {member.taskStatus && (
        <span className="text-xs text-[var(--color-text-secondary)]">
          {member.taskStatus}
        </span>
      )}

      {/* Cost */}
      {agent?.totalCostUSD && (
        <CostBadge cost={agent.totalCostUSD} />
      )}
    </Link>
  );
}

// ---- Stats row ----

function StatsRow({
  squad,
  agents,
}: {
  squad: Squad;
  agents: Map<string, AgentResponse>;
}) {
  const totalCost = useMemo(() => {
    let sum = 0;
    for (const m of squad.members) {
      const a = agents.get(m.name);
      if (a?.totalCostUSD) sum += parseFloat(a.totalCostUSD);
    }
    return sum > 0 ? sum.toFixed(4) : undefined;
  }, [squad.members, agents]);

  const readyCount = squad.members.filter((m) => m.ready).length;
  const runningCount = squad.members.filter(
    (m) => agents.get(m.name)?.status === "Running"
  ).length;

  const items: { label: string; value: string; highlight?: boolean }[] = [
    { label: "Members", value: String(squad.members.length) },
    { label: "Ready", value: `${readyCount}/${squad.members.length}` },
    { label: "Running", value: String(runningCount) },
    { label: "Total Cost", value: totalCost ? `$${totalCost}` : "—", highlight: !!totalCost },
  ];

  return (
    <div className="flex items-center rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] px-5 py-4 divide-x divide-[var(--color-border)]">
      {items.map((item) => (
        <div key={item.label} className="flex flex-col items-center gap-1 px-5 first:pl-0 last:pr-0">
          <span className="text-[10px] uppercase tracking-widest font-semibold text-[var(--color-text-muted)]">
            {item.label}
          </span>
          <span
            className={`font-mono font-semibold text-base tabular-nums ${
              item.highlight
                ? "text-[var(--color-brand-blue)]"
                : "text-[var(--color-text)]"
            }`}
          >
            {item.value}
          </span>
        </div>
      ))}
    </div>
  );
}

// ---- Main page ----

export default function SquadDetailPage() {
  const params = useParams<{ name: string }>();
  const name = params.name;
  const searchParams = useSearchParams();
  const namespace = searchParams.get("namespace") ?? "default";
  const router = useRouter();

  const [squad, setSquad] = useState<Squad | null>(null);
  const [agentMap, setAgentMap] = useState<Map<string, AgentResponse>>(new Map());
  const [loading, setLoading] = useState(true);
  const showLoading = useDelayedLoading(loading);
  const [notFound, setNotFound] = useState(false);
  const [actionInProgress, setActionInProgress] = useState<string | null>(null);

  const countdown = useOrphanCountdown(squad?.orphanedSince, squad?.orphanTTL);

  const fetchData = useCallback(async () => {
    try {
      const [squadData, agentsData] = await Promise.all([
        getSquad(name, namespace),
        listAgents(),
      ]);
      setSquad(squadData);
      const map = new Map<string, AgentResponse>();
      for (const a of agentsData.agents ?? []) {
        map.set(a.name, a);
      }
      setAgentMap(map);
      setNotFound(false);
    } catch (e: unknown) {
      if (e instanceof Error && e.message.toLowerCase().includes("not found")) {
        setNotFound(true);
      }
    } finally {
      setLoading(false);
    }
  }, [name, namespace]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // --- Actions ---

  async function handleWakeAll() {
    if (!squad) return;
    setActionInProgress("wake");
    try {
      await Promise.allSettled(
        squad.members.map((m) =>
          createAgent({ name: m.name, instructions: "wake", namespace })
        )
      );
      await fetchData();
    } finally {
      setActionInProgress(null);
    }
  }

  async function handleSleepAll() {
    if (!squad) return;
    setActionInProgress("sleep");
    try {
      await Promise.allSettled(
        squad.members.map((m) =>
          patchAgent(m.name, { lifecycle: "Sleep" }, namespace)
        )
      );
      await fetchData();
    } finally {
      setActionInProgress(null);
    }
  }

  async function handleDelete() {
    try {
      await deleteSquad(name, namespace);
      router.push("/squads");
    } catch {
      // non-critical, user will see the squad still listed
    }
  }

  // --- Render states ---

  if (showLoading) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex-1 overflow-y-auto p-6">
          <SkeletonTable />
        </div>
      </div>
    );
  }

  if (loading) return null;

  if (notFound || !squad) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex flex-1 flex-col items-center justify-center gap-4 text-center">
          <p className="text-lg font-medium text-[var(--color-text)]">Squad not found</p>
          <p className="text-sm text-[var(--color-text-secondary)]">
            The squad &quot;{name}&quot; does not exist or has been deleted.
          </p>
          <Link href="/squads">
            <Button variant="secondary" size="sm">
              Back to Squads
            </Button>
          </Link>
        </div>
      </div>
    );
  }

  const isOrphaned = squad.phase === "Orphaned";

  return (
    <div className="flex h-full flex-col">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: "easeOut" }}
        className="flex-1 overflow-y-auto p-6 space-y-6"
      >
        {/* Orphaned banner */}
        {isOrphaned && (
          <motion.div
            initial={{ opacity: 0, y: -6 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.25 }}
            className="flex items-center gap-3 rounded-[var(--radius-md)] border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm"
          >
            <AlertTriangle className="size-4 shrink-0 text-amber-400" />
            <span className="text-amber-300">
              This squad is orphaned — all member agents have completed or been removed.
              {countdown !== null && (
                <> Will be automatically deleted in <strong>{countdown}</strong>.</>
              )}
            </span>
          </motion.div>
        )}

        {/* Header: status + namespace + actions */}
        <div className="flex flex-wrap items-center gap-4">
          <StatusBadge status={squad.phase} />
          <span className="text-sm text-[var(--color-text-secondary)]">
            ns: <span className="font-medium text-[var(--color-text)]">{squad.namespace}</span>
          </span>
          {squad.breakUpRequested && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-amber-500/15 text-amber-300 border border-amber-500/30">
              Break-up pending
            </span>
          )}
          {squad.message && (
            <span className="text-xs text-[var(--color-text-muted)] italic truncate max-w-xs">
              {squad.message}
            </span>
          )}

          <div className="ml-auto flex items-center gap-2">
            <Button
              variant="secondary"
              size="sm"
              onClick={handleWakeAll}
              disabled={actionInProgress !== null}
            >
              <Zap className="size-3" data-icon="inline-start" />
              {actionInProgress === "wake" ? "Waking…" : "Wake all"}
            </Button>
            <Button
              variant="secondary"
              size="sm"
              onClick={handleSleepAll}
              disabled={actionInProgress !== null}
            >
              <Moon className="size-3" data-icon="inline-start" />
              {actionInProgress === "sleep" ? "Sleeping…" : "Sleep all"}
            </Button>
            <BreakUpButton squad={squad} onSuccess={fetchData} />
            <ConfirmDialog
              title={`Delete squad "${squad.name}"?`}
              description="This will delete the squad. Member agents will not be deleted but will no longer be grouped. This action cannot be undone."
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

        {/* Stats */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, ease: "easeOut", delay: 0.05 }}
        >
          <StatsRow squad={squad} agents={agentMap} />
        </motion.div>

        {/* Members */}
        <motion.section
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, ease: "easeOut", delay: 0.1 }}
        >
          <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
            Members ({squad.members.length})
          </h2>
          {squad.members.length > 0 ? (
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
              {squad.members.map((m) => (
                <MemberCard
                  key={m.name}
                  member={m}
                  agent={agentMap.get(m.name)}
                  namespace={namespace}
                />
              ))}
            </div>
          ) : (
            <p className="text-sm text-[var(--color-text-secondary)]">No members.</p>
          )}
        </motion.section>

        <motion.section
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, ease: "easeOut", delay: 0.2 }}
        >
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
              Topology
            </h2>
            <Link
              href={`/topology?squad=${squad.name}`}
              className="text-[11px] text-[var(--color-text-secondary)] hover:text-[var(--color-brand-blue)] transition-colors"
            >
              View in Topology →
            </Link>
          </div>
          <SquadTopologyGraph squad={squad} agents={Array.from(agentMap.values())} />
        </motion.section>

        {/* Info */}
        <motion.section
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, ease: "easeOut", delay: 0.25 }}
        >
          <div className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-5 space-y-3 transition-colors duration-150 hover:border-[var(--color-border-hover)]">
            <h3 className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)]">
              Squad Info
            </h3>
            <InfoRow label="Name" value={squad.name} />
            <InfoRow label="Namespace" value={squad.namespace} />
            <InfoRow label="Phase" value={squad.phase} />
            {squad.orphanTTL && <InfoRow label="Orphan TTL" value={squad.orphanTTL} />}
            {squad.orphanedSince && (
              <InfoRow
                label="Orphaned Since"
                value={new Date(squad.orphanedSince).toLocaleString()}
              />
            )}
            <InfoRow label="Created" value={new Date(squad.createdAt).toLocaleString()} />
          </div>
        </motion.section>
      </motion.div>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-[11px] uppercase tracking-wider text-[var(--color-text-muted)] shrink-0">
        {label}
      </span>
      <span className="text-[13px] truncate text-right text-[var(--color-text-secondary)]">
        {value}
      </span>
    </div>
  );
}
