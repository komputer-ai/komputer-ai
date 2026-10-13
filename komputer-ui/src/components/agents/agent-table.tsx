"use client";

import Link from "next/link";
import { ArrowDown, ArrowUp, ArrowUpDown, Check } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/kit/table";
import { StatusBadge } from "@/components/shared/status-badge";
import { CostBadge } from "@/components/shared/cost-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { AgentDeleteButton, agentKey } from "@/components/agents/agent-cards";
import { useSquads } from "@/hooks/use-squads";
import { toggleTableSort, type AgentTableSortKey } from "@/lib/agent-list-query";
import type { AgentResponse, Squad } from "@/lib/types";

type AgentTableProps = {
  agents: AgentResponse[];
  onDelete: (name: string, namespace: string, opts?: { recreatePod?: boolean }) => void;
  /** Current API sort string, e.g. "-created" or "cost". */
  sort: string;
  onSortChange: (sort: string) => void;
  selected?: Set<string>;
  onToggleSelect?: (key: string) => void;
};

function lifecycleLabel(lifecycle?: string): string {
  if (!lifecycle) return "Default";
  if (lifecycle === "AutoDelete") return "Auto Delete";
  return lifecycle;
}

const SORTABLE_COLUMNS: { key: AgentTableSortKey; label: string }[] = [
  { key: "name", label: "Name" },
  { key: "status", label: "Status" },
  { key: "model", label: "Model" },
  { key: "cost", label: "Cost" },
  { key: "created", label: "Created" },
  { key: "lastActive", label: "Last active" },
];

function SortableHead({
  label,
  columnKey,
  sort,
  onSortChange,
}: {
  label: string;
  columnKey: AgentTableSortKey;
  sort: string;
  onSortChange: (s: string) => void;
}) {
  const desc = sort.startsWith("-");
  const activeKey = desc ? sort.slice(1) : sort;
  const isActive = activeKey === columnKey;
  const Icon = !isActive ? ArrowUpDown : desc ? ArrowDown : ArrowUp;
  return (
    <TableHead>
      <button
        type="button"
        onClick={() => onSortChange(toggleTableSort(sort, columnKey))}
        aria-label={`Sort by ${label}`}
        className={`flex items-center gap-1 cursor-pointer select-none transition-colors hover:text-[var(--color-text)] ${
          isActive ? "text-[var(--color-text)]" : "text-[var(--color-text-secondary)]"
        }`}
      >
        {label}
        <Icon className="size-3" />
      </button>
    </TableHead>
  );
}

export function AgentTable({ agents, onDelete, sort, onSortChange, selected, onToggleSelect }: AgentTableProps) {
  const { squads } = useSquads();

  const agentSquadMap = new Map<string, Squad>();
  for (const squad of squads) {
    for (const member of squad.members) {
      agentSquadMap.set(member.name, squad);
    }
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          {onToggleSelect && <TableHead className="w-8" />}
          {SORTABLE_COLUMNS.map((c) => (
            <SortableHead key={c.key} label={c.label} columnKey={c.key} sort={sort} onSortChange={onSortChange} />
          ))}
          <TableHead>Lifecycle</TableHead>
          <TableHead className="w-10" />
        </TableRow>
      </TableHeader>
      <TableBody>
        <AnimatePresence initial={false}>
          {agents.map((agent, i) => {
            const key = agentKey(agent);
            const isSelected = !!selected?.has(key);
            const squad = agentSquadMap.get(agent.name);
            return (
              <motion.tr
                key={key}
                className="border-b transition-colors hover:bg-muted/50"
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -6 }}
                transition={{ duration: 0.2, delay: i * 0.01 }}
              >
                {onToggleSelect && (
                  <TableCell>
                    <button
                      type="button"
                      onClick={() => onToggleSelect(key)}
                      aria-label={isSelected ? `Deselect ${agent.name}` : `Select ${agent.name}`}
                      className={`flex items-center justify-center w-4.5 h-4.5 rounded border cursor-pointer ${
                        isSelected
                          ? "bg-[var(--color-brand-blue)] border-[var(--color-brand-blue)]"
                          : "border-[var(--color-border)]"
                      }`}
                    >
                      {isSelected && <Check className="size-3 text-white" strokeWidth={3} />}
                    </button>
                  </TableCell>
                )}
                <TableCell>
                  <Link
                    href={`/agents/${agent.name}?namespace=${agent.namespace}`}
                    className="font-medium text-[var(--color-brand-blue)] hover:underline"
                  >
                    {agent.name}
                  </Link>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-1.5">
                    <StatusBadge status={agent.status} size="sm" />
                    {agent.status === "Queued" && agent.queuePosition != null && agent.queuePosition > 0 && (
                      <span
                        className="text-xs px-1.5 py-0.5 rounded bg-yellow-500/20 text-yellow-300"
                        title={agent.queueReason}
                      >
                        #{agent.queuePosition}
                      </span>
                    )}
                  </div>
                </TableCell>
                <TableCell className="text-xs text-[var(--color-text-secondary)]">{agent.model}</TableCell>
                <TableCell>
                  <CostBadge cost={agent.totalCostUSD} />
                </TableCell>
                <TableCell>
                  <RelativeTime timestamp={agent.createdAt} />
                </TableCell>
                <TableCell>
                  {agent.lastActivityAt ? (
                    <RelativeTime timestamp={agent.lastActivityAt} />
                  ) : (
                    <span className="text-xs text-[var(--color-text-secondary)]">--</span>
                  )}
                </TableCell>
                <TableCell className="text-xs text-[var(--color-text-secondary)]">
                  <div className="flex items-center gap-1.5">
                    <span>{lifecycleLabel(agent.lifecycle)}</span>
                    {agent.storage?.ephemeral && (
                      <span
                        className="text-[10px] px-1.5 py-0.5 rounded bg-yellow-500/20 text-yellow-300"
                        title="Ephemeral workspace — no PVC, lost on pod restart"
                      >
                        Ephemeral
                      </span>
                    )}
                  </div>
                </TableCell>
                <TableCell>
                  <AgentDeleteButton
                    agentName={agent.name}
                    agentNamespace={agent.namespace}
                    squad={squad}
                    onConfirm={(opts) => onDelete(agent.name, agent.namespace, opts)}
                  />
                </TableCell>
              </motion.tr>
            );
          })}
        </AnimatePresence>
      </TableBody>
    </Table>
  );
}
