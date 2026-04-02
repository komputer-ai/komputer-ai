"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import { Trash2, Calendar, CheckCircle, DollarSign, Activity, Pencil, Check, X, Clock } from "lucide-react";

import { Button } from "@/components/kit/button";
import { Badge } from "@/components/kit/badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { CostBadge } from "@/components/shared/cost-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { Input } from "@/components/kit/input";
import { getSchedule, deleteSchedule, patchSchedule } from "@/lib/api";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { cronToHuman, formatCost } from "@/lib/utils";
import type { ScheduleResponse } from "@/lib/types";

function StatCard({
  label,
  value,
  icon: Icon,
}: {
  label: string;
  value: string;
  icon: React.ComponentType<{ className?: string }>;
}) {
  return (
    <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
      <div className="flex items-center gap-2 text-[var(--color-text-secondary)]">
        <Icon className="size-4" />
        <span className="text-xs uppercase tracking-wider">{label}</span>
      </div>
      <p className="mt-2 text-xl font-semibold text-[var(--color-text)]">
        {value}
      </p>
    </div>
  );
}

export default function ScheduleDetailPage() {
  const params = useParams<{ name: string }>();
  const name = params.name;
  const router = useRouter();

  const [schedule, setSchedule] = useState<ScheduleResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const showLoading = useDelayedLoading(loading);
  const [notFound, setNotFound] = useState(false);
  const [editingCron, setEditingCron] = useState(false);
  const [cronDraft, setCronDraft] = useState("");
  const [savingCron, setSavingCron] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const data = await getSchedule(name);
      setSchedule(data);
      setNotFound(false);
    } catch (e: unknown) {
      if (e instanceof Error && e.message.includes("not found")) {
        setNotFound(true);
      }
    } finally {
      setLoading(false);
    }
  }, [name]);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 10_000);
    return () => clearInterval(interval);
  }, [fetchData]);

  async function handleDelete() {
    try {
      await deleteSchedule(name);
      router.push("/schedules");
    } catch {
      // non-critical
    }
  }

  async function handleSaveCron() {
    const trimmed = cronDraft.trim();
    if (!trimmed || trimmed === schedule?.schedule) {
      setEditingCron(false);
      return;
    }
    setSavingCron(true);
    try {
      await patchSchedule(name, { schedule: trimmed });
      await fetchData();
      setEditingCron(false);
    } catch {
      // keep editing on error
    } finally {
      setSavingCron(false);
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

  if (notFound || !schedule) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex flex-1 flex-col items-center justify-center gap-4 text-center">
          <p className="text-lg font-medium text-[var(--color-text)]">
            Schedule not found
          </p>
          <p className="text-sm text-[var(--color-text-secondary)]">
            The schedule &quot;{name}&quot; does not exist or has been deleted.
          </p>
          <Link href="/schedules">
            <Button variant="secondary" size="sm">
              Back to Schedules
            </Button>
          </Link>
        </div>
      </div>
    );
  }

  const runCount = schedule.runCount ?? 0;
  const successfulRuns = schedule.successfulRuns ?? 0;
  const successRate =
    runCount > 0 ? `${Math.round((successfulRuns / runCount) * 100)}%` : "--";

  return (
    <div className="flex h-full flex-col">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: "easeOut" }}
        className="flex-1 overflow-y-auto p-6 space-y-8"
      >
        {/* Header bar */}
        <div className="flex flex-wrap items-center gap-3">
          <StatusBadge status={schedule.phase} />
          {schedule.timezone && (
            <Badge variant="secondary" className="text-[10px]">
              {schedule.timezone}
            </Badge>
          )}
          {schedule.autoDelete && (
            <Badge variant="secondary" className="text-[9px]">
              one-time
            </Badge>
          )}
          <div className="ml-auto">
            <ConfirmDialog
              title={`Delete ${schedule.name}?`}
              description="This will permanently delete this schedule. This action cannot be undone."
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

        {/* Cron expression — hero card */}
        <div className="relative overflow-hidden rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-surface)]">
          {/* Subtle gradient accent along top edge */}
          <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-[var(--color-brand-blue)]/40 to-transparent" />
          <div className="px-4 py-3">
            <AnimatePresence mode="wait">
              {editingCron ? (
                <motion.div
                  key="edit"
                  initial={{ opacity: 0, y: 2 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -2 }}
                  transition={{ duration: 0.12 }}
                  className="flex items-center gap-2"
                >
                  <Clock className="size-3.5 text-[var(--color-brand-blue)] shrink-0" />
                  <input
                    value={cronDraft}
                    onChange={(e) => setCronDraft(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") handleSaveCron();
                      if (e.key === "Escape") setEditingCron(false);
                    }}
                    className="flex-1 bg-transparent font-mono text-base font-medium tracking-wide text-[var(--color-text)] outline-none border-b border-[var(--color-brand-blue)] pb-0.5 caret-[var(--color-brand-blue)] placeholder:text-[var(--color-text-muted)]"
                    placeholder="* * * * *"
                    autoFocus
                    disabled={savingCron}
                  />
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 hover:bg-emerald-500/10"
                    onClick={handleSaveCron}
                    disabled={savingCron}
                  >
                    <Check className="size-3.5 text-emerald-400" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => setEditingCron(false)}
                    disabled={savingCron}
                  >
                    <X className="size-3.5 text-[var(--color-text-secondary)]" />
                  </Button>
                </motion.div>
              ) : (
                <motion.button
                  key="display"
                  initial={{ opacity: 0, y: 2 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -2 }}
                  transition={{ duration: 0.12 }}
                  type="button"
                  onClick={() => {
                    setCronDraft(schedule.schedule);
                    setEditingCron(true);
                  }}
                  className="group flex w-full items-center gap-2.5 text-left cursor-pointer"
                >
                  <Clock className="size-3.5 text-[var(--color-brand-blue)] shrink-0" />
                  <span className="font-mono text-base font-medium tracking-wide text-[var(--color-text)] group-hover:text-[var(--color-brand-blue)] transition-colors">
                    {schedule.schedule}
                  </span>
                  <span className="text-xs text-[var(--color-text-muted)]">
                    {cronToHuman(schedule.schedule)}
                  </span>
                  <Pencil className="size-3 text-[var(--color-text-muted)] opacity-0 group-hover:opacity-100 transition-opacity shrink-0 ml-auto" />
                </motion.button>
              )}
            </AnimatePresence>
          </div>
        </div>

        {/* Stats cards */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard
            label="Total Runs"
            value={String(runCount)}
            icon={Calendar}
          />
          <StatCard
            label="Success Rate"
            value={successRate}
            icon={CheckCircle}
          />
          <StatCard
            label="Total Cost"
            value={formatCost(schedule.totalCostUSD)}
            icon={DollarSign}
          />
          <StatCard
            label="Last Run Cost"
            value={formatCost(schedule.lastRunCostUSD)}
            icon={Activity}
          />
        </div>

        <div className="border-t border-[var(--color-border)]" />

        {/* Info section */}
        <section>
          <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
            Details
          </h2>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {/* Agent */}
            <div>
              <span className="text-xs text-[var(--color-text-secondary)]">
                Agent
              </span>
              <p className="mt-0.5">
                {schedule.agentName ? (
                  <Link
                    href={`/agents/${schedule.agentName}`}
                    className="text-sm font-medium text-[var(--color-brand-blue)] hover:underline"
                  >
                    {schedule.agentName}
                  </Link>
                ) : (
                  <span className="text-sm text-[var(--color-text-secondary)]">
                    --
                  </span>
                )}
              </p>
            </div>

            {/* Next run */}
            <div>
              <span className="text-xs text-[var(--color-text-secondary)]">
                Next Run
              </span>
              <p className="mt-0.5">
                {schedule.nextRunTime ? (
                  <RelativeTime timestamp={schedule.nextRunTime} />
                ) : (
                  <span className="text-sm text-[var(--color-text-secondary)]">
                    --
                  </span>
                )}
              </p>
            </div>

            {/* Last run */}
            <div>
              <span className="text-xs text-[var(--color-text-secondary)]">
                Last Run
              </span>
              <div className="mt-0.5 flex items-center gap-2">
                {schedule.lastRunTime ? (
                  <>
                    <RelativeTime timestamp={schedule.lastRunTime} />
                    {schedule.lastRunStatus && (
                      <StatusBadge status={schedule.lastRunStatus} size="sm" />
                    )}
                  </>
                ) : (
                  <span className="text-sm text-[var(--color-text-secondary)]">
                    --
                  </span>
                )}
              </div>
            </div>

            {/* Flags */}
            <div>
              <span className="text-xs text-[var(--color-text-secondary)]">
                Flags
              </span>
              <div className="mt-0.5 flex items-center gap-2">
                {schedule.autoDelete ? (
                  <Badge variant="secondary" className="text-[10px]">
                    Auto-delete
                  </Badge>
                ) : null}
                {schedule.keepAgents ? (
                  <Badge variant="secondary" className="text-[10px]">
                    Keep agents
                  </Badge>
                ) : null}
                {!schedule.autoDelete && !schedule.keepAgents && (
                  <span className="text-sm text-[var(--color-text-secondary)]">
                    --
                  </span>
                )}
              </div>
            </div>
          </div>
        </section>
      </motion.div>
    </div>
  );
}
