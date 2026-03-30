"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import { Info, ArrowUpDown } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/kit/card";
import { AnimatePresence, motion } from "framer-motion";
import { TopSpendersChart } from "@/components/costs/cost-charts";
import { EmptyState } from "@/components/shared/empty-state";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { useAgents } from "@/hooks/use-agents";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { formatCost } from "@/lib/utils";
import type { AgentResponse } from "@/lib/types";

type SortField = "totalCostUSD" | "lastTaskCostUSD" | "name";
type SortDir = "asc" | "desc";

function parseCost(cost?: string): number {
  if (!cost) return 0;
  const n = parseFloat(cost);
  return isNaN(n) ? 0 : n;
}

export default function CostsPage() {
  const { agents, loading, error } = useAgents();
  const showLoading = useDelayedLoading(loading);
  const [sortField, setSortField] = useState<SortField>("totalCostUSD");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  function toggleSort(field: SortField) {
    if (sortField === field) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortField(field);
      setSortDir("desc");
    }
  }

  const agentsWithCost = useMemo(
    () => agents.filter((a) => parseCost(a.totalCostUSD) > 0),
    [agents]
  );

  const totalCost = useMemo(
    () => agents.reduce((sum, a) => sum + parseCost(a.totalCostUSD), 0),
    [agents]
  );

  const avgCost = agents.length > 0 ? totalCost / agents.length : 0;

  const mostExpensive = useMemo(() => {
    if (agents.length === 0) return null;
    return agents.reduce<AgentResponse | null>((best, a) => {
      if (!best) return a;
      return parseCost(a.totalCostUSD) > parseCost(best.totalCostUSD)
        ? a
        : best;
    }, null);
  }, [agents]);

  const sorted = useMemo(() => {
    const list = [...agentsWithCost];
    list.sort((a, b) => {
      let cmp: number;
      if (sortField === "name") {
        cmp = a.name.localeCompare(b.name);
      } else {
        cmp = parseCost(a[sortField]) - parseCost(b[sortField]);
      }
      return sortDir === "asc" ? cmp : -cmp;
    });
    return list;
  }, [agentsWithCost, sortField, sortDir]);

  return (
    <div className="flex h-full flex-col">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.1, ease: "easeOut" }}
        className="flex-1 overflow-y-auto p-6 space-y-6"
      >
        {showLoading ? (
          <SkeletonTable />
        ) : loading ? (
          null
        ) : error ? (
          <div className="rounded-lg border border-red-400/20 bg-red-400/5 p-4 text-sm text-red-400">
            Failed to load agents: {error}
          </div>
        ) : agents.length === 0 ? (
          <EmptyState
            title="No agents yet"
            description="Create agents to start tracking costs."
          />
        ) : (
          <>
            {/* Total cost card */}
            <Card className="border-[var(--color-border)] bg-[var(--color-surface)]">
              <CardContent className="py-6 text-center">
                <p className="text-4xl font-bold text-[var(--color-brand-blue)] font-mono">
                  ${totalCost.toFixed(4)}
                </p>
                <p className="mt-1 text-sm text-[var(--color-text-secondary)]">
                  Total across {agents.length} agent{agents.length !== 1 ? "s" : ""}
                </p>
              </CardContent>
            </Card>

            {/* Stats row */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <Card className="border-[var(--color-border)] bg-[var(--color-surface)]">
                <CardHeader>
                  <CardTitle className="text-xs uppercase tracking-wider text-[var(--color-text-secondary)]">
                    Avg Cost / Agent
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-lg font-semibold text-[var(--color-text)] font-mono">
                    ${avgCost.toFixed(4)}
                  </p>
                </CardContent>
              </Card>

              <Card className="border-[var(--color-border)] bg-[var(--color-surface)]">
                <CardHeader>
                  <CardTitle className="text-xs uppercase tracking-wider text-[var(--color-text-secondary)]">
                    Most Expensive
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  {mostExpensive && parseCost(mostExpensive.totalCostUSD) > 0 ? (
                    <>
                      <p className="text-lg font-semibold text-[var(--color-text)] font-mono">
                        {formatCost(mostExpensive.totalCostUSD)}
                      </p>
                      <p className="text-xs text-[var(--color-text-secondary)] truncate">
                        {mostExpensive.name}
                      </p>
                    </>
                  ) : (
                    <p className="text-sm text-[var(--color-text-secondary)]">
                      No cost data
                    </p>
                  )}
                </CardContent>
              </Card>

              <Card className="border-[var(--color-border)] bg-[var(--color-surface)]">
                <CardHeader>
                  <CardTitle className="text-xs uppercase tracking-wider text-[var(--color-text-secondary)]">
                    Agents with Cost
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-lg font-semibold text-[var(--color-text)] font-mono">
                    {agentsWithCost.length}
                    <span className="text-sm font-normal text-[var(--color-text-secondary)]">
                      {" "}/ {agents.length}
                    </span>
                  </p>
                </CardContent>
              </Card>
            </div>

            {/* Top spenders chart */}
            <Card className="border-[var(--color-border)] bg-[var(--color-surface)]">
              <CardHeader>
                <CardTitle className="text-sm font-medium text-[var(--color-text)]">
                  Top Spenders
                </CardTitle>
              </CardHeader>
              <CardContent>
                <TopSpendersChart agents={agents} />
              </CardContent>
            </Card>

            {/* Cost breakdown cards */}
            {agentsWithCost.length > 0 && (
              <div>
                <div className="mb-4 flex items-center justify-between">
                  <h3 className="text-sm font-medium text-[var(--color-text)]">
                    Cost Breakdown
                  </h3>
                  <div className="flex gap-1">
                    {(["name", "totalCostUSD", "lastTaskCostUSD"] as SortField[]).map((field) => (
                      <button
                        key={field}
                        className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-border)]/50 transition-colors"
                        onClick={() => toggleSort(field)}
                      >
                        {field === "name" ? "Name" : field === "totalCostUSD" ? "Total" : "Last Task"}
                        <ArrowUpDown className="size-3" />
                      </button>
                    ))}
                  </div>
                </div>
                <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-2.5">
                  <AnimatePresence initial={false}>
                    {sorted.map((agent, i) => (
                      <motion.div
                        key={agent.name}
                        initial={{ opacity: 0, scale: 0.97 }}
                        animate={{ opacity: 1, scale: 1 }}
                        exit={{ opacity: 0, scale: 0.97 }}
                        transition={{ duration: 0.2, delay: i * 0.03 }}
                      >
                        <Link href={`/agents/${agent.name}`} className="block group">
                          <div className="relative overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] transition-all duration-200 group-hover:border-[var(--color-border-hover)] group-hover:shadow-[0_0_20px_rgba(63,133,217,0.06),0_0_40px_rgba(139,92,246,0.04)]">
                            <span className="absolute top-2.5 right-2.5 block w-2 h-2 rounded-full bg-[var(--color-brand-blue)]" />
                            <div className="h-full flex flex-col p-3">
                              <span className="text-[13px] font-semibold text-[var(--color-text)] truncate block">{agent.name}</span>
                              <div className="mt-auto">
                                <p className="text-base font-bold text-[var(--color-brand-blue)] font-mono">
                                  {formatCost(agent.totalCostUSD)}
                                </p>
                                <p className="text-[10px] text-[var(--color-text-secondary)] font-mono mt-0.5">
                                  last: {formatCost(agent.lastTaskCostUSD)}
                                </p>
                              </div>
                            </div>
                          </div>
                        </Link>
                      </motion.div>
                    ))}
                  </AnimatePresence>
                </div>
              </div>
            )}

            {/* Note banner */}
            <div className="flex items-start gap-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
              <Info className="mt-0.5 size-4 shrink-0 text-[var(--color-text-secondary)]" />
              <p className="text-xs text-[var(--color-text-secondary)]">
                Cost history over time is not yet available. Showing current snapshot.
              </p>
            </div>
          </>
        )}
      </motion.div>
    </div>
  );
}
