"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { usePageRefresh } from "@/components/layout/app-shell";
import {
  Bot,
  DollarSign,
  Building2,
  CalendarClock,
  Wrench,
  Brain,
  KeyRound,
  Plug,
  Users,
  ArrowRight,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";

import { Card, CardContent } from "@/components/kit/card";
import {
  listAgents,
  listOffices,
  listSchedules,
  listSkills,
  listMemories,
  listSecrets,
  listConnectors,
  listSquads,
} from "@/lib/api";
import { formatCost, formatRelativeTime } from "@/lib/utils";
import type {
  AgentResponse,
  OfficeResponse,
  ScheduleResponse,
  SkillResponse,
  MemoryResponse,
  SecretResponse,
  ConnectorResponse,
  Squad,
  SkillListResponse,
  MemoryListResponse,
  SecretListResponse,
  ConnectorListResponse,
  SquadListResponse,
} from "@/lib/types";
import { SuggestedTasks } from "@/components/dashboard/suggested-tasks";
import { PersonalAgentPrompt } from "@/components/dashboard/personal-agent-prompt";

// --- Animated number ---

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

// --- Stat card ---

function StatCard({
  icon,
  label,
  value,
  delay = 0,
  iconClassName,
  breakdown,
  href,
}: {
  icon: React.ReactNode;
  label: string;
  value: number | string;
  delay?: number;
  iconClassName?: string;
  breakdown?: { color: string; count: number; label: string }[];
  href?: string;
}) {
  const inner = (
    <Card className="h-[88px] bg-[var(--color-surface)] border-[var(--color-border)] ring-0 hover:shadow-[0_4px_16px_rgba(var(--shadow-color),var(--shadow-strength)),inset_0_1px_0_var(--color-border-light)] hover:border-[var(--color-border-hover)]">
      <CardContent className="flex h-full items-center gap-4 py-4">
        <div
          className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${iconClassName ?? "bg-[var(--color-brand-blue)]/10 text-[var(--color-brand-blue)]"}`}
        >
          {icon}
        </div>
        <div className="min-w-0">
          <p className="text-xs text-[var(--color-text-secondary)]">{label}</p>
          <div className="flex items-baseline gap-3 min-w-0">
            <p className="text-2xl font-semibold text-[var(--color-text)] tabular-nums shrink-0">
              <AnimatedNumber value={value} />
            </p>
            {breakdown && breakdown.some((b) => b.count > 0) && (
              <div className="flex items-center gap-x-2 overflow-hidden">
                {breakdown.map((b) =>
                  b.count > 0 ? (
                    <span key={b.label} className="inline-flex items-center gap-1 text-xs text-[var(--color-text)] whitespace-nowrap">
                      <span className="size-1.5 rounded-full shrink-0" style={{ backgroundColor: b.color }} />
                      {b.count} {b.label}
                    </span>
                  ) : null
                )}
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      whileHover={{ y: -2, transition: { duration: 0.15 } }}
      transition={{ duration: 0.3, delay }}
      className={`cursor-pointer shrink-0 w-[calc((100%-48px)/4)] min-w-[220px]`}
    >
      {href ? <Link href={href}>{inner}</Link> : inner}
    </motion.div>
  );
}

// --- Scrollable stats row ---

function StatsScrollRow({ children }: { children: React.ReactNode }) {
  const rowRef = useRef<HTMLDivElement>(null);
  const [showLeft, setShowLeft] = useState(false);
  const [showRight, setShowRight] = useState(false);

  const updateScrollState = useCallback(() => {
    const el = rowRef.current;
    if (!el) return;
    setShowLeft(el.scrollLeft > 4);
    setShowRight(el.scrollLeft + el.clientWidth < el.scrollWidth - 4);
  }, []);

  useEffect(() => {
    const el = rowRef.current;
    if (!el) return;
    updateScrollState();
    el.addEventListener("scroll", updateScrollState, { passive: true });
    const ro = new ResizeObserver(updateScrollState);
    ro.observe(el);
    return () => {
      el.removeEventListener("scroll", updateScrollState);
      ro.disconnect();
    };
  }, [updateScrollState]);

  const scrollBy = (direction: "left" | "right") => {
    const el = rowRef.current;
    if (!el) return;
    const delta = direction === "right" ? 280 : -280;
    const target = Math.max(0, Math.min(el.scrollWidth - el.clientWidth, el.scrollLeft + delta));
    el.scrollTo({ left: target, behavior: "smooth" });
  };

  return (
    // -mb-2 cancels half of the row's pb-6 in the outer space-y-6 stack so
    // the visual gap to the next section stays comfortable while the row
    // still reserves room internally for hover-shadow render below the cards.
    <div className="relative min-w-0 -mb-2">
      {/* Left fade + chevron */}
      {showLeft && (
        <>
          <div
            className="pointer-events-none absolute left-0 top-0 bottom-0 w-8 z-10"
            style={{
              background: "linear-gradient(to right, var(--color-bg), transparent)",
            }}
          />
          <button
            onClick={() => scrollBy("left")}
            className="absolute left-1 top-1/2 -translate-y-1/2 z-20 flex h-7 w-7 items-center justify-center rounded-full bg-[var(--color-surface-raised)] border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-text)] shadow-sm transition-colors"
            aria-label="Scroll left"
          >
            <ChevronLeft className="size-3.5" />
          </button>
        </>
      )}

      {/* Right fade + chevron */}
      {showRight && (
        <>
          <div
            className="pointer-events-none absolute right-0 top-0 bottom-0 w-8 z-10"
            style={{
              background: "linear-gradient(to left, var(--color-bg), transparent)",
            }}
          />
          <button
            onClick={() => scrollBy("right")}
            className="absolute right-1 top-1/2 -translate-y-1/2 z-20 flex h-7 w-7 items-center justify-center rounded-full bg-[var(--color-surface-raised)] border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-text)] shadow-sm transition-colors"
            aria-label="Scroll right"
          >
            <ChevronRight className="size-3.5" />
          </button>
        </>
      )}

      <div
        ref={rowRef}
        className="flex gap-4 overflow-x-auto pt-2 pb-6"
        style={{ scrollbarWidth: "none" }}
      >
        <style>{`.stats-row::-webkit-scrollbar { display: none; }`}</style>
        {children}
      </div>
    </div>
  );
}

// --- Main ---

export default function DashboardPage() {
  const [agents, setAgents] = useState<AgentResponse[]>([]);
  const [offices, setOffices] = useState<OfficeResponse[]>([]);
  const [schedules, setSchedules] = useState<ScheduleResponse[]>([]);
  const [skills, setSkills] = useState<SkillResponse[]>([]);
  const [memories, setMemories] = useState<MemoryResponse[]>([]);
  const [secrets, setSecrets] = useState<SecretResponse[]>([]);
  const [connectors, setConnectors] = useState<ConnectorResponse[]>([]);
  const [squads, setSquads] = useState<Squad[]>([]);
  const [loading, setLoading] = useState(true);
  const showLoading = useDelayedLoading(loading);
  // True while the prompt bar has an active streaming session — used to
  // subtly blur the rest of the dashboard so the bubbles draw focus.
  const [bubbleSessionActive, setBubbleSessionActive] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const [agentsRes, officesRes, schedulesRes, skillsRes, memoriesRes, secretsRes, connectorsRes, squadsRes] =
        await Promise.all([
          listAgents(),
          listOffices(),
          listSchedules(),
          listSkills().catch(() => ({ skills: [] })),
          listMemories().catch(() => ({ memories: [] })),
          listSecrets().catch(() => ({ secrets: [] })),
          listConnectors().catch(() => ({ connectors: [] })),
          listSquads().catch(() => ({ squads: [] })),
        ]);
      setAgents(agentsRes.agents ?? []);
      setOffices(officesRes.offices ?? []);
      setSchedules(schedulesRes.schedules ?? []);
      setSkills((skillsRes as SkillListResponse).skills ?? []);
      setMemories((memoriesRes as MemoryListResponse).memories ?? []);
      setSecrets((secretsRes as SecretListResponse).secrets ?? []);
      setConnectors((connectorsRes as ConnectorListResponse).connectors ?? []);
      setSquads((squadsRes as SquadListResponse).squads ?? []);
    } catch {
      // silently fail
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  usePageRefresh(fetchData);

  // --- Computed stats ---

  const totalCost = agents
    .reduce((sum, a) => sum + (parseFloat(a.totalCostUSD ?? "0") || 0), 0)
    .toFixed(4);

  // Agent status breakdown
  const agentsByStatus = {
    Running: agents.filter((a) => a.status === "Running").length,
    Sleeping: agents.filter((a) => a.status === "Sleeping").length,
    Pending: agents.filter((a) => a.status === "Pending").length,
    Failed: agents.filter((a) => a.status === "Failed").length,
    Succeeded: agents.filter((a) => a.status === "Succeeded").length,
  };

  // Office status breakdown
  const officesByPhase = {
    InProgress: offices.filter((o) => o.phase === "InProgress").length,
    Complete: offices.filter((o) => o.phase === "Complete").length,
    Error: offices.filter((o) => o.phase === "Error").length,
  };

  // Schedule status breakdown
  const schedulesByPhase = {
    Active: schedules.filter((s) => s.phase === "Active").length,
    Suspended: schedules.filter((s) => s.phase === "Suspended").length,
    Error: schedules.filter((s) => s.phase === "Error").length,
  };

  // Connector OAuth breakdown
  const oauthConnectors = connectors.filter((c) => c.authType === "oauth" && c.oauthStatus);
  const connectorsByOAuth = {
    connected: oauthConnectors.filter((c) => c.oauthStatus === "connected").length,
    pending: oauthConnectors.filter((c) => c.oauthStatus === "pending").length,
  };

  // Squad phase breakdown
  const squadsByPhase = {
    Running: squads.filter((s) => s.phase === "Running").length,
    Pending: squads.filter((s) => s.phase === "Pending").length,
    Orphaned: squads.filter((s) => s.phase === "Orphaned").length,
    Failed: squads.filter((s) => s.phase === "Failed").length,
  };

  // Running tasks (agents with InProgress task)
  const runningTasks = agents.filter((a) => a.taskStatus === "InProgress");

  // Recent agents — fill 2 rows at the widest breakpoint (6 cols × 2 = 12)
  const recentAgents = [...agents]
    .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
    .slice(0, 12);

  // Top cost agents (sorted by total cost, top 5)
  const topCostAgents = [...agents]
    .filter((a) => parseFloat(a.totalCostUSD ?? "0") > 0)
    .sort((a, b) => parseFloat(b.totalCostUSD ?? "0") - parseFloat(a.totalCostUSD ?? "0"))
    .slice(0, 5);

  return (
    <div className="flex flex-col h-full">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: "easeOut" }}
        className="flex-1 overflow-y-auto p-6 space-y-6"
      >
        {/* Hero — blurred while a streaming session is active so the bubbles
            draw focus. Inline style + an explicit transition declaration so
            the property animation runs reliably regardless of class-toggle
            timing. */}
        <motion.div
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4, ease: "easeOut" }}
          className="text-center py-2"
          style={{
            filter: bubbleSessionActive ? "blur(2px)" : "blur(0px)",
            opacity: bubbleSessionActive ? 0.6 : 1,
            transition: "filter 400ms ease, opacity 400ms ease",
          }}
        >
          <h1 className="text-xl font-semibold text-[var(--color-text)]">
            Welcome to <span className="bg-gradient-to-r from-[var(--color-text-secondary)] via-[var(--color-text)] to-[var(--color-text-secondary)] bg-clip-text text-transparent animate-shine">Komputer.AI</span>
          </h1>
          <p className="mt-1 text-sm text-[var(--color-text-secondary)]">
            Your AI agent fleet management at scale.
          </p>
        </motion.div>

        <PersonalAgentPrompt onSessionActiveChange={setBubbleSessionActive} />

        {/* Everything below the prompt fades into a subtle blur while a
            streaming session is active. Sidebar (outside this tree) stays
            crisp; prompt bar + bubbles are unaffected because they're not
            inside this wrapper. */}
        <div
          className="space-y-6"
          style={{
            filter: bubbleSessionActive ? "blur(2px)" : "blur(0px)",
            opacity: bubbleSessionActive ? 0.6 : 1,
            transition: "filter 400ms ease, opacity 400ms ease",
            pointerEvents: bubbleSessionActive ? "none" : "auto",
          }}
        >

        {/* Stats row — horizontally scrollable */}
        <StatsScrollRow>
          <StatCard
            icon={<Bot className="size-5" />}
            label="Total Agents"
            value={agents.length}
            delay={0}
            href="/agents"
            breakdown={[
              { color: "#34D399", count: agentsByStatus.Running, label: "Running" },
              { color: "#FBBF24", count: agentsByStatus.Sleeping, label: "Sleeping" },
              { color: "#FBBF24", count: agentsByStatus.Pending, label: "Pending" },
              { color: "#F87171", count: agentsByStatus.Failed, label: "Failed" },
              { color: "#34D399", count: agentsByStatus.Succeeded, label: "Succeeded" },
            ]}
          />
          <StatCard
            icon={<DollarSign className="size-5" />}
            label="Total Cost"
            value={totalCost}
            delay={0.04}
            iconClassName="bg-[var(--color-brand-violet)]/10 text-[var(--color-brand-violet)]"
          />
          <StatCard
            icon={<Building2 className="size-5" />}
            label="Offices"
            value={offices.length}
            delay={0.08}
            href="/offices"
            breakdown={[
              { color: "#34D399", count: officesByPhase.InProgress, label: "In Progress" },
              { color: "#34D399", count: officesByPhase.Complete, label: "Complete" },
              { color: "#F87171", count: officesByPhase.Error, label: "Error" },
            ]}
          />
          <StatCard
            icon={<CalendarClock className="size-5" />}
            label="Schedules"
            value={schedules.length}
            delay={0.12}
            href="/schedules"
            iconClassName="bg-[var(--color-brand-violet)]/10 text-[var(--color-brand-violet)]"
            breakdown={[
              { color: "#34D399", count: schedulesByPhase.Active, label: "Active" },
              { color: "#FBBF24", count: schedulesByPhase.Suspended, label: "Suspended" },
              { color: "#F87171", count: schedulesByPhase.Error, label: "Error" },
            ]}
          />
          <StatCard
            icon={<Plug className="size-5" />}
            label="Connectors"
            value={connectors.length}
            delay={0.28}
            href="/connectors"
            iconClassName="bg-[var(--color-brand-violet)]/10 text-[var(--color-brand-violet)]"
            breakdown={[
              { color: "#34D399", count: connectorsByOAuth.connected, label: "Connected" },
              { color: "#FBBF24", count: connectorsByOAuth.pending, label: "Pending" },
            ]}
          />
          <StatCard
            icon={<Users className="size-5" />}
            label="Squads"
            value={squads.length}
            delay={0.32}
            href="/squads"
            breakdown={[
              { color: "#34D399", count: squadsByPhase.Running, label: "Running" },
              { color: "#FBBF24", count: squadsByPhase.Pending, label: "Pending" },
              { color: "#F87171", count: squadsByPhase.Orphaned, label: "Orphaned" },
              { color: "#F87171", count: squadsByPhase.Failed, label: "Failed" },
            ]}
          />
          <StatCard
            icon={<Wrench className="size-5" />}
            label="Skills"
            value={skills.length}
            delay={0.16}
            href="/skills"
          />
          <StatCard
            icon={<Brain className="size-5" />}
            label="Memories"
            value={memories.length}
            delay={0.20}
            href="/memories"
            iconClassName="bg-[var(--color-brand-violet)]/10 text-[var(--color-brand-violet)]"
          />
          <StatCard
            icon={<KeyRound className="size-5" />}
            label="Secrets"
            value={secrets.length}
            delay={0.24}
            href="/secrets"
          />
        </StatsScrollRow>

        {/* Suggested Tasks (above when empty) */}
        {!loading && agents.length === 0 && <SuggestedTasks />}

        {/* Recent Agents */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.25 }}
        >
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-muted)]">Recent Agents</h3>
            <Link href="/agents" className="text-[10px] text-[var(--color-brand-blue)] hover:underline flex items-center gap-0.5">
              View all <ArrowRight className="size-2.5" />
            </Link>
          </div>
          {showLoading ? (
            <div className="grid grid-cols-6 gap-2.5">
              {Array.from({ length: 6 }).map((_, i) => (
                <div key={i} className="h-[130px] rounded-[var(--radius-md)] bg-[var(--color-surface)] animate-pulse" />
              ))}
            </div>
          ) : recentAgents.length === 0 ? (
            <div className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] py-8 text-center text-sm text-[var(--color-text-muted)]">
              No agents yet. Create one to get started.
            </div>
          ) : (
            <div className="grid grid-cols-6 gap-2.5">
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
                    transition={{ duration: 0.2, delay: 0.25 + i * 0.03 }}
                  >
                    <Link href={`/agents/${agent.name}?namespace=${agent.namespace}`} className="block group">
                      <div
                        className="relative overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] transition-all duration-200 group-hover:border-[var(--color-border-hover)] hover:shadow-[0_4px_16px_rgba(var(--shadow-color),var(--shadow-strength)),inset_0_1px_0_var(--color-border-light)]"
                      >
                        <span className="absolute top-2.5 right-2.5 block w-2 h-2 rounded-full" style={{ backgroundColor: color }} />
                        <div className="h-full flex flex-col p-3">
                          <span className="text-[13px] font-semibold text-[var(--color-text)] truncate pr-4">{agent.name}</span>
                          <div className="mt-4 space-y-1">
                            <div className="flex items-center justify-between">
                              <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">status</span>
                              <span className="text-[11px] font-medium text-[var(--color-text-secondary)]">{agent.status}</span>
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
        </motion.div>

        {/* Running tasks + Top cost — side by side, fill remaining space */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Running tasks */}
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.3 }}
            className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-3"
          >
            <h3 className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-muted)] mb-2">
              Running Tasks
            </h3>
            {showLoading ? (
              <div className="space-y-1.5">
                {[0, 1].map((i) => (
                  <div key={i} className="h-10 rounded-md bg-[var(--color-bg)] animate-pulse" />
                ))}
              </div>
            ) : runningTasks.length === 0 ? (
              <p className="text-xs text-[var(--color-text-muted)] py-2 text-center">
                No tasks running right now
              </p>
            ) : (
              <div className="space-y-1.5 max-h-36 overflow-y-auto">
                {runningTasks.map((agent) => (
                  <Link
                    key={agent.name}
                    href={`/agents/${agent.name}?namespace=${agent.namespace}`}
                    className="flex items-start gap-2.5 rounded-md bg-[var(--color-bg)] p-2 transition-colors hover:bg-[var(--color-bg-subtle)] group"
                  >
                    <span className="mt-1 size-2 shrink-0 rounded-full bg-[#34D399] animate-pulse" />
                    <div className="min-w-0 flex-1">
                      <p className="text-xs font-medium text-[var(--color-text)] truncate group-hover:text-[var(--color-brand-blue-light)]">
                        {agent.name}
                      </p>
                      {agent.lastTaskMessage && (
                        <p className="text-[11px] text-[var(--color-text-secondary)] truncate mt-0.5">
                          {agent.lastTaskMessage}
                        </p>
                      )}
                    </div>
                    <span className="text-[10px] text-[var(--color-text-muted)] shrink-0">
                      {agent.namespace}
                    </span>
                  </Link>
                ))}
              </div>
            )}
          </motion.div>

          {/* Top cost agents */}
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.35 }}
            className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-3"
          >
            <h3 className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-muted)] mb-2">
              Top Cost Agents
            </h3>
            {showLoading ? (
              <div className="space-y-1.5">
                {[0, 1].map((i) => (
                  <div key={i} className="h-9 rounded-md bg-[var(--color-bg)] animate-pulse" />
                ))}
              </div>
            ) : topCostAgents.length === 0 ? (
              <p className="text-xs text-[var(--color-text-muted)] py-2 text-center">
                No cost data yet
              </p>
            ) : (
              <div className="space-y-1 max-h-36 overflow-y-auto">
                {topCostAgents.map((agent, i) => {
                  const cost = parseFloat(agent.totalCostUSD ?? "0");
                  const maxCost = parseFloat(topCostAgents[0].totalCostUSD ?? "1");
                  const pct = maxCost > 0 ? (cost / maxCost) * 100 : 0;

                  return (
                    <Link
                      key={agent.name}
                      href={`/agents/${agent.name}?namespace=${agent.namespace}`}
                      className="flex items-center gap-2.5 rounded-md bg-[var(--color-bg)] p-2 transition-colors hover:bg-[var(--color-bg-subtle)] group"
                    >
                      <span className="text-[11px] font-mono text-[var(--color-text-muted)] w-4 text-right shrink-0">
                        {i + 1}
                      </span>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center justify-between mb-0.5">
                          <span className="text-xs font-medium text-[var(--color-text)] truncate group-hover:text-[var(--color-brand-blue-light)]">
                            {agent.name}
                          </span>
                          <span className="text-[11px] font-mono text-[var(--color-text)] shrink-0 ml-2">
                            {formatCost(agent.totalCostUSD)}
                          </span>
                        </div>
                        <div className="h-0.5 rounded-full bg-[var(--color-border)] overflow-hidden">
                          <motion.div
                            className="h-full rounded-full bg-gradient-to-r from-[var(--color-brand-blue)] to-[var(--color-brand-violet)]"
                            initial={{ width: 0 }}
                            animate={{ width: `${pct}%` }}
                            transition={{ duration: 0.6, delay: 0.4 + i * 0.1, ease: "easeOut" }}
                          />
                        </div>
                      </div>
                    </Link>
                  );
                })}
              </div>
            )}
          </motion.div>
        </div>

        {/* Suggested Tasks (bottom when agents exist) */}
        {!loading && agents.length > 0 && <SuggestedTasks />}
        </div>
      </motion.div>
    </div>
  );
}
