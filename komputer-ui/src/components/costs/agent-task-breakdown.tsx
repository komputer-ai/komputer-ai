"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { ArrowUpDown, RefreshCw, Flame } from "lucide-react";
import { Tooltip } from "@/components/kit/tooltip";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/kit/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/kit/select";
import { getAgentCostBreakdown } from "@/lib/api";
import type { AgentResponse, CostBreakdownResponse, TaskBreakdown } from "@/lib/types";

type SortKey = "index" | "costUSD" | "durationMs" | "inputTokens";

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  const s = ms / 1000;
  if (s < 60) return `${s.toFixed(1)}s`;
  const m = Math.floor(s / 60);
  return `${m}m ${Math.round(s % 60)}s`;
}

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}m`;
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return String(n);
}

export function AgentTaskBreakdown({ agents }: { agents: AgentResponse[] }) {
  const router = useRouter();
  const [selectedAgent, setSelectedAgent] = useState<string>("");
  const [breakdown, setBreakdown] = useState<CostBreakdownResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sortKey, setSortKey] = useState<SortKey>("costUSD");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");

  const agentsWithCost = agents
    .filter((a) => parseFloat(a.totalCostUSD || "0") > 0)
    .filter((a, i, arr) => arr.findIndex((b) => b.name === a.name) === i)
    .sort((a, b) => parseFloat(b.totalCostUSD || "0") - parseFloat(a.totalCostUSD || "0"));

  const fetchBreakdown = async (name: string, refresh = false) => {
    setError(null);
    setLoading(true);
    try {
      const data = await getAgentCostBreakdown(name, undefined, refresh);
      setBreakdown(data);
    } catch {
      setError("Failed to load cost breakdown");
      setBreakdown(null);
    } finally {
      setLoading(false);
    }
  };

  const handleSelect = async (name: string) => {
    setSelectedAgent(name);
    await fetchBreakdown(name);
  };

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("desc");
    }
  };

  const sortedTasks = breakdown
    ? [...breakdown.tasks].sort((a, b) => {
        const cmp = (a[sortKey] ?? 0) - (b[sortKey] ?? 0);
        return sortDir === "asc" ? cmp : -cmp;
      })
    : [];

  const mostExpensiveTask = breakdown
    ? breakdown.tasks.reduce<TaskBreakdown | null>(
        (best, t) => (!best || t.costUSD > best.costUSD ? t : best),
        null
      )
    : null;

  return (
    <Card className="border-[var(--color-border)] bg-[var(--color-surface)]">
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium text-[var(--color-text)]">
            Task Breakdown
          </CardTitle>
          <div className="flex items-center gap-2">
            <Select value={selectedAgent} onValueChange={handleSelect}>
              <SelectTrigger className="w-48">
                <SelectValue placeholder="Select an agent..." />
              </SelectTrigger>
              <SelectContent className="min-w-[280px] right-0 left-auto">
                {agentsWithCost.map((a) => {
                  const cost = parseFloat(a.totalCostUSD || "0");
                  return (
                    <SelectItem key={a.name} value={a.name}>
                      <span className="truncate">{a.name}</span>
                      <span className="ml-auto pl-3 shrink-0 font-mono text-[10px] text-[var(--color-text-muted)] tabular-nums">
                        ${cost.toFixed(4)}
                      </span>
                    </SelectItem>
                  );
                })}
              </SelectContent>
            </Select>
            {selectedAgent && (
              <Tooltip content="Reprocess events (clears cache)" side="left">
                <button
                  type="button"
                  onClick={() => fetchBreakdown(selectedAgent, true)}
                  disabled={loading}
                  className="flex size-8 items-center justify-center rounded-md text-[var(--color-text-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface-hover)] transition-colors cursor-pointer disabled:opacity-50"
                >
                  <RefreshCw className={`size-3.5 ${loading ? "animate-spin" : ""}`} />
                </button>
              </Tooltip>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {!selectedAgent ? (
          <p className="py-8 text-center text-sm text-[var(--color-text-secondary)]">
            Select an agent to see per-task cost breakdown.
          </p>
        ) : loading ? (
          <p className="py-8 text-center text-sm text-[var(--color-text-secondary)]">
            Processing event history...
          </p>
        ) : error ? (
          <p className="py-8 text-center text-sm text-red-400">{error}</p>
        ) : breakdown && breakdown.tasks.length === 0 ? (
          <p className="py-8 text-center text-sm text-[var(--color-text-secondary)]">
            No completed tasks found for this agent.
          </p>
        ) : breakdown ? (
          <div className="space-y-4">
            {/* Summary stats */}
            <div className="flex gap-4 text-xs text-[var(--color-text-secondary)]">
              <span>{breakdown.taskCount} tasks</span>
              <span className="font-mono text-[var(--color-brand-blue)]">
                ${breakdown.totalCost.toFixed(4)} total
              </span>
              {mostExpensiveTask && (
                <span>
                  most expensive:{" "}
                  <span className="font-mono text-[var(--color-text)]">
                    ${mostExpensiveTask.costUSD.toFixed(4)}
                  </span>
                </span>
              )}
              {breakdown.cachedAt && (
                <span className="ml-auto text-[var(--color-text-muted)]">
                  cached {new Date(breakdown.cachedAt).toLocaleTimeString()}
                </span>
              )}
            </div>

            {/* Sort buttons */}
            <div className="flex gap-1">
              {(
                [
                  ["index", "Order"],
                  ["costUSD", "Cost"],
                  ["durationMs", "Duration"],
                  ["inputTokens", "Tokens"],
                ] as [SortKey, string][]
              ).map(([key, label]) => (
                <button
                  key={key}
                  onClick={() => toggleSort(key)}
                  className={`inline-flex items-center gap-1 rounded px-2 py-1 text-xs transition-colors cursor-pointer ${
                    sortKey === key
                      ? "text-[var(--color-brand-blue)] bg-[var(--color-brand-blue-glow)]"
                      : "text-[var(--color-text-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-border)]/50"
                  }`}
                >
                  {label}
                  <ArrowUpDown className="size-3" />
                </button>
              ))}
            </div>

            {/* Task list — max 10 visible, scroll for the rest */}
            <div className="space-y-1 max-h-[440px] overflow-y-auto">
              <AnimatePresence mode="popLayout">
                {sortedTasks.map((task, i) => {
                  const isMostExpensive = mostExpensiveTask?.index === task.index;
                  return (
                    <motion.div
                      key={task.index}
                      layout
                      initial={{ opacity: 0, x: -8 }}
                      animate={{ opacity: 1, x: 0 }}
                      exit={{ opacity: 0, x: 8 }}
                      transition={{ duration: 0.2, delay: i * 0.02 }}
                      onClick={() => {
                        const params = new URLSearchParams();
                        params.set("taskFrom", task.startedAt);
                        if (task.completedAt) params.set("taskTo", task.completedAt);
                        // Pass event count so we fetch just enough + a small buffer
                        params.set("taskEvents", String(task.events?.length ?? 50));
                        router.push(`/agents/${selectedAgent}?${params.toString()}`);
                      }}
                      className={`flex items-center gap-3 rounded-md px-3 py-2 text-xs transition-colors cursor-pointer hover:bg-[var(--color-surface-hover)] ${
                        isMostExpensive
                          ? "border border-amber-500/20 bg-amber-500/5"
                          : ""
                      }`}
                    >
                      <span className="w-6 text-right font-mono text-[var(--color-text-muted)]">
                        #{task.index + 1}
                      </span>
                      <span className="flex-1 min-w-0 truncate text-[var(--color-text)]">
                        {task.steer && <span className="mr-1.5 text-[var(--color-brand-violet)]">steer</span>}
                        {task.instruction || "—"}
                      </span>
                      {isMostExpensive && (
                        <span className="shrink-0 inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-[10px] font-medium text-amber-400">
                          <Flame className="size-2.5" />
                          most expensive
                        </span>
                      )}
                      <span className="shrink-0 font-mono text-[var(--color-brand-blue)] w-20 text-right">
                        ${task.costUSD.toFixed(4)}
                      </span>
                      <span className="shrink-0 w-16 text-right text-[var(--color-text-secondary)]">
                        {task.durationMs > 0 ? formatDuration(task.durationMs) : "—"}
                      </span>
                      <span className="shrink-0 w-16 text-right font-mono text-[var(--color-text-muted)]">
                        {task.inputTokens > 0 ? formatTokens(task.inputTokens + task.outputTokens) : "—"}
                      </span>
                      <span className="shrink-0 w-12 text-right text-[var(--color-text-muted)]">
                        {task.turns > 0 ? `${task.turns}t` : "—"}
                      </span>
                    </motion.div>
                  );
                })}
              </AnimatePresence>
            </div>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}
