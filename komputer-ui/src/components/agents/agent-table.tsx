"use client";

import Link from "next/link";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowUp, ArrowUpDown, Bot, Check, CheckCircle2, Clock, HardDriveDownload, Loader2, Moon, Power, Skull, Timer, Users, Zap } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { RelativeTime } from "@/components/shared/relative-time";
import { AgentDeleteButton, agentKey } from "@/components/agents/agent-cards";
import { useSquads } from "@/hooks/use-squads";
import { sleepAgent, wakeAgent } from "@/lib/api";
import { toggleTableSort, type AgentTableSortKey } from "@/lib/agent-list-query";
import { shortModelLabel } from "@/lib/model-utils";
import { cn, formatCost } from "@/lib/utils";
import type { AgentResponse, Squad } from "@/lib/types";

type AgentTableProps = {
  agents: AgentResponse[];
  onDelete: (name: string, namespace: string, opts?: { recreatePod?: boolean }) => void;
  /** Current API sort string, e.g. "-created" or "cost". */
  sort: string;
  onSortChange: (sort: string) => void;
  selected?: Set<string>;
  onToggleSelect?: (key: string) => void;
  /** Called after a row's Sleep/Wake action succeeds, to refresh the list. */
  onChanged?: () => void;
};

// Same palette and icons as the cards view so both views read the same.
const STATUS: Record<string, { color: string; icon: typeof Bot }> = {
  Running: { color: "var(--color-status-running)", icon: Zap },
  Sleeping: { color: "var(--color-status-sleeping)", icon: Moon },
  Pending: { color: "var(--color-status-pending)", icon: Clock },
  Queued: { color: "var(--color-status-pending)", icon: Clock },
  Failed: { color: "var(--color-status-error)", icon: Skull },
  Succeeded: { color: "var(--color-status-success)", icon: CheckCircle2 },
};
const DEFAULT_STATUS = { color: "var(--color-text-muted)", icon: Bot };

const MODEL_FAMILY_COLOR: Record<string, string> = {
  opus: "var(--color-brand-violet-light)",
  sonnet: "var(--color-brand-blue-light)",
  haiku: "var(--color-status-success)",
  fable: "var(--color-status-pending)",
};

type Column = {
  key: AgentTableSortKey | "lifecycle";
  label: string;
  sortable: boolean;
  className?: string;
};

// Responsive: secondary columns drop out on narrow screens instead of forcing
// a horizontal scroll (which would also break the sticky header).
const COLUMNS: Column[] = [
  { key: "name", label: "Agent", sortable: true },
  { key: "status", label: "Status", sortable: true, className: "w-[120px]" },
  { key: "model", label: "Model", sortable: true, className: "w-[130px] hidden md:table-cell" },
  { key: "cost", label: "Cost", sortable: true, className: "w-[110px] text-right" },
  { key: "created", label: "Created", sortable: true, className: "w-[100px] hidden lg:table-cell" },
  { key: "lastActive", label: "Last active", sortable: true, className: "w-[110px] hidden lg:table-cell" },
  { key: "lifecycle", label: "Lifecycle", sortable: false, className: "w-[140px] hidden xl:table-cell" },
];

function SortHeader({ column, sort, onSortChange }: { column: Column; sort: string; onSortChange: (s: string) => void }) {
  const desc = sort.startsWith("-");
  const active = column.sortable && (desc ? sort.slice(1) : sort) === column.key;
  const alignRight = column.className?.includes("text-right");
  return (
    <th
      scope="col"
      aria-sort={active ? (desc ? "descending" : "ascending") : undefined}
      className={cn("h-9 px-3 text-left align-middle font-normal whitespace-nowrap", column.className)}
    >
      {column.sortable ? (
        <button
          type="button"
          onClick={() => onSortChange(toggleTableSort(sort, column.key as AgentTableSortKey))}
          className={cn(
            "group/sort inline-flex h-4 items-center gap-1 align-middle leading-none cursor-pointer select-none text-[10px] uppercase tracking-wider transition-colors",
            alignRight && "flex-row-reverse",
            active ? "text-[var(--color-text)]" : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]",
          )}
        >
          {column.label}
          {active ? (
            <motion.span
              initial={false}
              animate={{ rotate: desc ? 180 : 0 }}
              transition={{ type: "spring", stiffness: 400, damping: 28 }}
              className="inline-flex text-[var(--color-brand-blue-light)]"
            >
              <ArrowUp className="size-3" />
            </motion.span>
          ) : (
            <ArrowUpDown className="size-3 opacity-0 transition-opacity group-hover/sort:opacity-60" />
          )}
        </button>
      ) : (
        <span className="inline-flex h-4 items-center align-middle text-[10px] uppercase leading-none tracking-wider text-[var(--color-text-muted)]">{column.label}</span>
      )}
    </th>
  );
}

// Wake for sleeping agents, Sleep for idle running ones; nothing otherwise
// (busy agents can't be slept, and ephemeral ones have no workspace to keep).
function SleepWakeButton({ agent, onChanged }: { agent: AgentResponse; onChanged?: () => void }) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const busy = agent.taskStatus === "InProgress" || agent.taskStatus === "Compacting";
  const action =
    agent.status === "Sleeping" ? "wake" : agent.status === "Running" && !busy && !agent.storage?.ephemeral ? "sleep" : null;
  if (!action) return null;

  const run = async () => {
    setPending(true);
    setError(null);
    try {
      if (action === "wake") await wakeAgent(agent.name, agent.namespace);
      else await sleepAgent(agent.name, agent.namespace);
      onChanged?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : `Failed to ${action}`);
    } finally {
      setPending(false);
    }
  };

  const Icon = pending ? Loader2 : action === "wake" ? Power : Moon;
  return (
    <button
      type="button"
      onClick={run}
      disabled={pending}
      title={error ?? (action === "wake" ? `Wake ${agent.name}` : `Sleep ${agent.name} now`)}
      className={cn(
        "inline-flex h-6 items-center gap-1 whitespace-nowrap rounded-md border px-2 text-[11px] font-medium transition-colors cursor-pointer disabled:cursor-default",
        error
          ? "border-[var(--color-status-error)]/50 text-[var(--color-status-error)]"
          : action === "wake"
            ? "border-[var(--color-status-running)]/30 bg-[var(--color-status-running)]/10 text-[var(--color-status-running)] hover:bg-[var(--color-status-running)]/20"
            : "border-[var(--color-border)] text-[var(--color-text-secondary)] opacity-0 group-hover:opacity-100 focus-visible:opacity-100 hover:border-[var(--color-border-hover)] hover:text-[var(--color-text)]",
      )}
    >
      <Icon className={cn("size-3", pending && "animate-spin")} />
      {action === "wake" ? "Wake" : "Sleep"}
    </button>
  );
}

function lifecyclePill(agent: AgentResponse) {
  if (agent.lifecycle === "Sleep") return { icon: Moon, label: "Sleeps after task" };
  if (agent.lifecycle === "AutoDelete") return { icon: Timer, label: "Deletes after task" };
  return null;
}

export function AgentTable({ agents, onDelete, sort, onSortChange, selected, onToggleSelect, onChanged }: AgentTableProps) {
  const router = useRouter();
  const { squads } = useSquads();
  const selectionMode = !!selected && selected.size > 0;

  const agentSquadMap = new Map<string, Squad>();
  for (const squad of squads) {
    for (const member of squad.members) {
      agentSquadMap.set(member.name, squad);
    }
  }

  // Cost bars are scaled to the most expensive agent currently loaded.
  const maxCost = agents.reduce((m, a) => Math.max(m, parseFloat(a.totalCostUSD ?? "") || 0), 0);

  return (
    <div className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] overflow-clip">
      <table className="w-full table-fixed border-separate border-spacing-0 text-sm">
        <thead className="sticky top-0 z-10">
          <tr className="bg-[var(--color-bg-subtle)]/85 backdrop-blur-md">
            {onToggleSelect && <th scope="col" className="w-11 border-b border-[var(--color-border)]" aria-label="Select" />}
            {COLUMNS.map((c) => (
              <SortHeader
                key={c.key}
                column={{ ...c, className: cn(c.className, "border-b border-[var(--color-border)]") }}
                sort={sort}
                onSortChange={onSortChange}
              />
            ))}
            <th scope="col" className="w-[110px] border-b border-[var(--color-border)]" aria-label="Actions" />
          </tr>
        </thead>
        <tbody>
          <AnimatePresence initial={true}>
            {agents.map((agent, i) => {
              const key = agentKey(agent);
              const isSelected = !!selected?.has(key);
              const squad = agentSquadMap.get(agent.name);
              const status = STATUS[agent.status] ?? DEFAULT_STATUS;
              const StatusIcon = status.icon;
              const busy = agent.taskStatus === "InProgress" || agent.taskStatus === "Compacting";
              const href = `/agents/${agent.name}?namespace=${agent.namespace}`;
              const cost = parseFloat(agent.totalCostUSD ?? "");
              const hasCost = !isNaN(cost);
              const costShare = hasCost && maxCost > 0 ? cost / maxCost : 0;
              const family = agent.model?.match(/(opus|sonnet|haiku|fable)/)?.[1];
              const lifecycle = lifecyclePill(agent);
              const last = i === agents.length - 1;
              const cell = cn("px-3 align-middle", !last && "border-b border-[var(--color-border-light)]");

              const onRowClick = (e: React.MouseEvent) => {
                // React bubbles clicks from portals (the delete dialog and its
                // backdrop) through the row, so ignore anything outside the row's
                // own DOM, and let real controls handle their own clicks.
                const target = e.target as HTMLElement;
                if (!e.currentTarget.contains(target) || target.closest("a,button")) return;
                if (selectionMode && onToggleSelect) {
                  onToggleSelect(key);
                } else if (e.metaKey || e.ctrlKey) {
                  window.open(href, "_blank");
                } else {
                  router.push(href);
                }
              };

              return (
                <motion.tr
                  key={key}
                  layout="position"
                  onClick={onRowClick}
                  initial={{ opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, transition: { duration: 0.12 } }}
                  transition={{
                    duration: 0.22,
                    ease: "easeOut",
                    // Stagger only the first screenful so a 50-row page doesn't crawl in.
                    delay: Math.min(i % 50, 12) * 0.018,
                    layout: { type: "spring", stiffness: 500, damping: 40 },
                  }}
                  className={cn(
                    "group relative h-12 cursor-pointer transition-colors duration-150",
                    isSelected
                      ? "bg-[var(--color-brand-blue)]/[0.08]"
                      : "hover:bg-[var(--color-surface-hover)]",
                  )}
                >
                  {onToggleSelect && (
                    <td className={cn(cell, "relative pl-4 pr-0")}>
                      {/* Selection edge */}
                      <motion.span
                        aria-hidden
                        initial={false}
                        animate={{ scaleY: isSelected ? 1 : 0, opacity: isSelected ? 1 : 0 }}
                        transition={{ duration: 0.18, ease: "easeOut" }}
                        className="absolute left-0 top-1.5 bottom-1.5 w-[3px] origin-center rounded-r-full bg-[var(--color-brand-blue)]"
                      />
                      <button
                        type="button"
                        onClick={() => onToggleSelect(key)}
                        aria-label={isSelected ? `Deselect ${agent.name}` : `Select ${agent.name}`}
                        aria-pressed={isSelected}
                        className={cn(
                          "flex size-4 items-center justify-center rounded-[4px] border cursor-pointer transition-all duration-150",
                          isSelected
                            ? "border-[var(--color-brand-blue)] bg-[var(--color-brand-blue)] opacity-100"
                            : cn(
                                "border-[var(--color-text-muted)] hover:border-[var(--color-text-secondary)]",
                                selectionMode ? "opacity-100" : "opacity-0 group-hover:opacity-100 focus-visible:opacity-100",
                              ),
                        )}
                      >
                        {isSelected && (
                          <Check className="size-3 text-white" strokeWidth={3} style={{ animation: "agent-check-pop 180ms ease-out" }} />
                        )}
                      </button>
                    </td>
                  )}

                  {/* Agent */}
                  <td className={cell}>
                    <div className="flex min-w-0 items-center gap-2.5">
                      <span
                        className="relative flex size-7 shrink-0 items-center justify-center rounded-md transition-transform duration-150 group-hover:scale-105"
                        style={{ backgroundColor: `color-mix(in srgb, ${status.color} 14%, transparent)` }}
                      >
                        <StatusIcon className="size-3.5" style={{ color: status.color }} />
                        {squad && (
                          <span
                            title={`In squad: ${squad.name}`}
                            className="absolute -bottom-1 -right-1 flex size-3.5 items-center justify-center rounded-full border border-[var(--color-surface)] bg-[var(--color-brand-violet)]"
                          >
                            <Users className="size-2 text-white" strokeWidth={3} />
                          </span>
                        )}
                      </span>
                      <div className="flex min-w-0 flex-col">
                        <Link
                          href={href}
                          title={agent.name}
                          className="truncate text-[13px] font-semibold leading-tight text-[var(--color-text)] transition-colors group-hover:text-[var(--color-brand-blue-light)]"
                        >
                          {agent.name}
                        </Link>
                        <span className="truncate text-[11px] leading-tight text-[var(--color-text-muted)]">
                          {agent.namespace}
                          {agent.storage?.ephemeral && (
                            <span title="Ephemeral workspace — no PVC, lost on pod restart">
                              {" · "}
                              <HardDriveDownload className="inline size-2.5 -translate-y-px text-[var(--color-status-pending)]" /> ephemeral
                            </span>
                          )}
                        </span>
                      </div>
                    </div>
                  </td>

                  {/* Status */}
                  <td className={cell}>
                    <span className="inline-flex items-center gap-2 text-xs text-[var(--color-text-secondary)]">
                      <span className="relative flex size-2">
                        {busy && (
                          <span
                            className="absolute inline-flex size-full animate-ping rounded-full opacity-60"
                            style={{ backgroundColor: status.color }}
                          />
                        )}
                        <span className="relative inline-flex size-2 rounded-full" style={{ backgroundColor: status.color }} />
                      </span>
                      {busy ? <span className="text-[var(--color-text)]">Working</span> : agent.status}
                      {agent.status === "Queued" && agent.queuePosition != null && agent.queuePosition > 0 && (
                        <span
                          className="rounded px-1 py-px font-mono text-[10px] text-[var(--color-status-pending)] bg-[var(--color-status-pending)]/15"
                          title={agent.queueReason}
                        >
                          #{agent.queuePosition}
                        </span>
                      )}
                    </span>
                  </td>

                  {/* Model */}
                  <td className={cn(cell, "hidden md:table-cell")}>
                    {agent.model && (
                      <span
                        title={agent.model}
                        className="inline-flex max-w-full items-center gap-1.5 rounded-full border border-[var(--color-border)] bg-[var(--color-bg-subtle)] px-2 py-0.5 font-mono text-[11px] text-[var(--color-text-secondary)]"
                      >
                        <span
                          className="size-1.5 shrink-0 rounded-full"
                          style={{ backgroundColor: (family && MODEL_FAMILY_COLOR[family]) || "var(--color-text-muted)" }}
                        />
                        <span className="truncate">{shortModelLabel(agent.model)}</span>
                      </span>
                    )}
                  </td>

                  {/* Cost */}
                  <td className={cn(cell, "text-right")}>
                    {hasCost ? (
                      <span className="inline-flex flex-col items-end gap-1" title={formatCost(agent.totalCostUSD)}>
                        <span className="font-mono text-xs tabular-nums text-[var(--color-text)]">
                          ${cost.toFixed(2)}
                        </span>
                        <span className="h-[3px] w-16 overflow-hidden rounded-full bg-[var(--color-border-light)]">
                          <motion.span
                            className="block h-full rounded-full bg-gradient-to-r from-[var(--color-brand-blue)] to-[var(--color-brand-violet)]"
                            initial={{ width: 0 }}
                            animate={{ width: `${Math.max(costShare * 100, costShare > 0 ? 4 : 0)}%` }}
                            transition={{ duration: 0.5, ease: "easeOut", delay: 0.1 }}
                          />
                        </span>
                      </span>
                    ) : (
                      <span className="text-xs text-[var(--color-text-muted)]">—</span>
                    )}
                  </td>

                  {/* Created */}
                  <td className={cn(cell, "hidden lg:table-cell")}>
                    <RelativeTime timestamp={agent.createdAt} />
                  </td>

                  {/* Last active */}
                  <td className={cn(cell, "hidden lg:table-cell")}>
                    {agent.lastActivityAt ? (
                      <RelativeTime timestamp={agent.lastActivityAt} />
                    ) : (
                      <span className="text-xs text-[var(--color-text-muted)]">Never</span>
                    )}
                  </td>

                  {/* Lifecycle */}
                  <td className={cn(cell, "hidden xl:table-cell")}>
                    {lifecycle ? (
                      <span className="inline-flex items-center gap-1 whitespace-nowrap text-xs text-[var(--color-text-secondary)]">
                        <lifecycle.icon className="size-3" />
                        {lifecycle.label}
                      </span>
                    ) : (
                      <span className="text-xs text-[var(--color-text-muted)]">Default</span>
                    )}
                  </td>

                  {/* Actions */}
                  <td className={cn(cell, "pr-3 text-right")}>
                    <span className="inline-flex items-center justify-end gap-1.5">
                      <SleepWakeButton agent={agent} onChanged={onChanged} />
                      <span className="inline-flex opacity-0 transition-opacity duration-150 group-hover:opacity-100 focus-within:opacity-100">
                      <AgentDeleteButton
                        agentName={agent.name}
                        agentNamespace={agent.namespace}
                        squad={squad}
                        onConfirm={(opts) => onDelete(agent.name, agent.namespace, opts)}
                      />
                      </span>
                    </span>
                  </td>
                </motion.tr>
              );
            })}
          </AnimatePresence>
        </tbody>
      </table>
    </div>
  );
}
