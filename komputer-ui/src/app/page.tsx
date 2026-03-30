"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { Bot, DollarSign, Building2, CalendarClock } from "lucide-react";

import { Card, CardContent } from "@/components/kit/card";
import { Button } from "@/components/kit/button";
import { StatusBadge } from "@/components/shared/status-badge";
import { listAgents, listOffices, listSchedules, checkHealth } from "@/lib/api";
import { formatCost, formatRelativeTime } from "@/lib/utils";
import type { AgentResponse, OfficeResponse, ScheduleResponse } from "@/lib/types";

function AnimatedNumber({ value }: { value: number | string }) {
  const [display, setDisplay] = useState(0);
  const target = typeof value === "string" ? parseFloat(value) || 0 : value;
  const isDecimal = typeof value === "string";

  useEffect(() => {
    const duration = 600;
    const start = performance.now();
    let raf: number;

    function tick(now: number) {
      const elapsed = now - start;
      const progress = Math.min(elapsed / duration, 1);
      // ease-out cubic
      const eased = 1 - Math.pow(1 - progress, 3);
      setDisplay(eased * target);
      if (progress < 1) {
        raf = requestAnimationFrame(tick);
      }
    }

    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [target]);

  if (isDecimal) {
    return <span>${display.toFixed(4)}</span>;
  }
  return <span>{Math.round(display)}</span>;
}

interface StatCardProps {
  icon: React.ReactNode;
  label: string;
  value: number | string;
  delay?: number;
  iconClassName?: string;
}

function StatCard({ icon, label, value, delay = 0, iconClassName }: StatCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      whileHover={{ y: -2, transition: { duration: 0.15 } }}
      transition={{ duration: 0.3, delay }}
      className="cursor-pointer"
    >
      <Card className="bg-[var(--color-surface)] border-[var(--color-border)] ring-0 hover:shadow-[0_4px_16px_rgba(0,0,0,0.2),inset_0_1px_0_var(--color-border-light)] hover:border-[var(--color-border-hover)]">
        <CardContent className="flex items-center gap-4 py-4">
          <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${iconClassName ?? "bg-[var(--color-brand-blue)]/10 text-[var(--color-brand-blue)]"}`}>
            {icon}
          </div>
          <div className="min-w-0">
            <p className="text-xs text-[var(--color-text-secondary)]">
              {label}
            </p>
            <p className="text-2xl font-semibold text-[var(--color-text)] tabular-nums">
              <AnimatedNumber value={value} />
            </p>
          </div>
        </CardContent>
      </Card>
    </motion.div>
  );
}

export default function DashboardPage() {
  const [agents, setAgents] = useState<AgentResponse[]>([]);
  const [offices, setOffices] = useState<OfficeResponse[]>([]);
  const [schedules, setSchedules] = useState<ScheduleResponse[]>([]);
  const [healthy, setHealthy] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);
  const showLoading = useDelayedLoading(loading);

  const fetchData = useCallback(async () => {
    try {
      const [agentsRes, officesRes, schedulesRes] = await Promise.all([
        listAgents(),
        listOffices(),
        listSchedules(),
      ]);
      setAgents(agentsRes.agents ?? []);
      setOffices(officesRes.offices ?? []);
      setSchedules(schedulesRes.schedules ?? []);
    } catch {
      // silently fail, health check will show API status
    } finally {
      setLoading(false);
    }
  }, []);

  const pollHealth = useCallback(async () => {
    try {
      const ok = await checkHealth();
      setHealthy(ok);
    } catch {
      setHealthy(false);
    }
  }, []);

  // initial load + 10s activity refresh
  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 10_000);
    return () => clearInterval(interval);
  }, [fetchData]);

  // health poll every 30s
  useEffect(() => {
    pollHealth();
    const interval = setInterval(pollHealth, 30_000);
    return () => clearInterval(interval);
  }, [pollHealth]);

  // computed stats
  const activeAgents = agents.filter((a) => a.status === "Running").length;
  const totalCost = agents
    .reduce((sum, a) => sum + (parseFloat(a.totalCostUSD ?? "0") || 0), 0)
    .toFixed(4);
  const officesInProgress = offices.filter((o) => o.phase === "InProgress").length;
  const activeSchedules = schedules.filter((s) => s.phase === "Active").length;

  // 10 most recent agents
  const recentAgents = [...agents]
    .sort(
      (a, b) =>
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    )
    .slice(0, 10);

  return (
    <div className="flex flex-col h-full">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: "easeOut" }}
        className="flex-1 overflow-y-auto p-6 space-y-6"
      >
        {/* Health indicator */}
        <div className="fixed bottom-4 right-4 z-50 flex items-center gap-2">
          <span
            className={`inline-block h-2.5 w-2.5 rounded-full ${
              healthy === true
                ? "bg-green-400"
                : healthy === false
                ? "bg-red-400"
                : "bg-[var(--color-text-secondary)]"
            }`}
          />
          <span className="text-[10px] text-[var(--color-text-secondary)]">
            API
          </span>
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard
            icon={<Bot className="size-5" />}
            label="Active Agents"
            value={activeAgents}
            delay={0}
          />
          <StatCard
            icon={<DollarSign className="size-5" />}
            label="Total Cost"
            value={totalCost}
            delay={0.05}
            iconClassName="bg-[var(--color-brand-violet)]/10 text-[var(--color-brand-violet)]"
          />
          <StatCard
            icon={<Building2 className="size-5" />}
            label="Offices In Progress"
            value={officesInProgress}
            delay={0.1}
          />
          <StatCard
            icon={<CalendarClock className="size-5" />}
            label="Active Schedules"
            value={activeSchedules}
            delay={0.15}
            iconClassName="bg-[var(--color-brand-violet)]/10 text-[var(--color-brand-violet)]"
          />
        </div>

        {/* Recent Agents */}
        <div>
          <h2 className="text-sm font-medium text-[var(--color-text-secondary)] mb-3">
            Recent Agents
          </h2>
          {showLoading ? (
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-2.5">
              {Array.from({ length: 10 }).map((_, i) => (
                <div
                  key={i}
                  className="h-[150px] rounded-[var(--radius-md)] bg-[var(--color-surface)] animate-pulse"
                />
              ))}
            </div>
          ) : loading ? (
            null
          ) : recentAgents.length === 0 ? (
            <Card className="bg-[var(--color-surface)] border-[var(--color-border)] ring-0">
              <CardContent className="py-8 text-center text-sm text-[var(--color-text-secondary)]">
                No agents yet. Create one to get started.
              </CardContent>
            </Card>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-2.5">
              {recentAgents.map((agent, i) => {
                const statusColors: Record<string, string> = {
                  Running: "#34D399", Sleeping: "#FBBF24", Pending: "#FBBF24",
                  Failed: "#F87171", Succeeded: "#34D399",
                };
                const color = statusColors[agent.status] ?? "#8899A6";
                return (
                  <motion.div
                    key={agent.name}
                    initial={{ opacity: 0, scale: 0.97 }}
                    animate={{ opacity: 1, scale: 1 }}
                    transition={{ duration: 0.2, delay: i * 0.03 }}
                  >
                    <Link href={`/agents/${agent.name}`} className="block group">
                      <div className="relative overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] transition-all duration-200 group-hover:border-[var(--color-border-hover)] group-hover:shadow-[0_0_20px_rgba(63,133,217,0.06),0_0_40px_rgba(139,92,246,0.04)]">
                        <span className="absolute top-2.5 right-2.5 block w-2 h-2 rounded-full" style={{ backgroundColor: color }} />
                        <div className="h-full flex flex-col p-3">
                          <span className="text-[13px] font-semibold text-[var(--color-text)] truncate pr-4">{agent.name}</span>
                          <div className="mt-4 space-y-1">
                            <div className="flex items-center justify-between">
                              <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">status</span>
                              <span className="text-[11px] font-medium text-[var(--color-text-secondary)]">{agent.status}</span>
                            </div>
                            <div className="flex items-center justify-between">
                              <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">model</span>
                              <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text-secondary)] truncate ml-2">{agent.model?.replace("claude-", "")}</span>
                            </div>
                            <div className="flex items-center justify-between">
                              <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">cost</span>
                              <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text)]">{formatCost(agent.totalCostUSD)}</span>
                            </div>
                            <div className="flex items-center justify-between">
                              <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">age</span>
                              <span className="text-[11px] text-[var(--color-text-secondary)]">{formatRelativeTime(agent.createdAt)}</span>
                            </div>
                          </div>
                        </div>
                      </div>
                    </Link>
                  </motion.div>
                );
              })}
            </div>
          )}
        </div>
      </motion.div>
    </div>
  );
}
