"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import { Trash2, Calendar, CheckCircle, DollarSign, Activity, Pencil, Check, X, Clock, Play, Pause, Save } from "lucide-react";

import { Button } from "@/components/kit/button";
import { Badge } from "@/components/kit/badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { Label } from "@/components/kit/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/kit/select";
import { getSchedule, deleteSchedule, patchSchedule, triggerSchedule, listAgents } from "@/lib/api";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { cronToHuman, formatCost, fmtDuration } from "@/lib/utils";
import type { ScheduleResponse, PatchScheduleRequest } from "@/lib/types";
import {
  AgentFieldsForm,
  type AgentFormValues,
  agentFormValuesFromScheduleSpec,
  buildScheduleAgentSpec,
} from "@/components/agents/agent-fields-form";

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
  const searchParams = useSearchParams();
  const scheduleNs = searchParams.get("namespace") || undefined;

  const [schedule, setSchedule] = useState<ScheduleResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const showLoading = useDelayedLoading(loading);
  const [notFound, setNotFound] = useState(false);
  const [editingCron, setEditingCron] = useState(false);
  const [cronDraft, setCronDraft] = useState("");
  const [savingCron, setSavingCron] = useState(false);
  const [editingInstructions, setEditingInstructions] = useState(false);
  const [instructionsDraft, setInstructionsDraft] = useState("");
  const [savingInstructions, setSavingInstructions] = useState(false);
  const [triggering, setTriggering] = useState(false);
  const [triggerMessage, setTriggerMessage] = useState<string | null>(null);
  // Inline editor for the "Details" panel — covers agent target, timezone,
  // flags, suspended toggle, and the inline ScheduleAgentSpec template.
  const [editingDetails, setEditingDetails] = useState(false);
  const [savingDetails, setSavingDetails] = useState(false);
  const [detailsDraft, setDetailsDraft] = useState<{
    agentName: string;
    timezone: string;
    autoDelete: boolean;
    keepAgents: boolean;
    suspended: boolean;
    /** Inline agent template, edited with the same form the create dialog uses. */
    agent: AgentFormValues;
  } | null>(null);
  const [availableAgents, setAvailableAgents] = useState<{ name: string; namespace?: string }[]>([]);

  const fetchData = useCallback(async () => {
    try {
      const data = await getSchedule(name, scheduleNs);
      setSchedule(data);
      setNotFound(false);
    } catch (e: unknown) {
      if (e instanceof Error && e.message.includes("not found")) {
        setNotFound(true);
      }
    } finally {
      setLoading(false);
    }
  }, [name, scheduleNs]);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 10_000);
    return () => clearInterval(interval);
  }, [fetchData]);

  // Load the agent list lazily — only when the user opens the editor.
  useEffect(() => {
    if (!editingDetails || availableAgents.length > 0) return;
    listAgents()
      .then((res) =>
        setAvailableAgents(
          (res.agents ?? []).map((a) => ({ name: a.name, namespace: a.namespace }))
        )
      )
      .catch(() => {
        /* non-critical */
      });
  }, [editingDetails, availableAgents.length]);

  function openDetailsEditor() {
    if (!schedule) return;
    setDetailsDraft({
      agentName: schedule.agentName ?? "",
      timezone: schedule.timezone ?? "",
      autoDelete: !!schedule.autoDelete,
      keepAgents: !!schedule.keepAgents,
      suspended: !!schedule.suspended,
      // Mirror the create dialog's defaults (worker / Sleep) when the schedule
      // has no inline template yet, so switching to "new agent" starts sane.
      agent: schedule.agent
        ? agentFormValuesFromScheduleSpec(schedule.agent, schedule.namespace)
        : agentFormValuesFromScheduleSpec({ role: "worker", lifecycle: "Sleep" }, schedule.namespace),
    });
    setEditingDetails(true);
  }

  async function handleSaveDetails() {
    if (!schedule || !detailsDraft) return;
    const patch: PatchScheduleRequest = {};
    const draftAgent = detailsDraft.agentName.trim();
    const currentAgent = schedule.agentName ?? "";
    if (draftAgent !== currentAgent) patch.agentName = draftAgent;
    if (detailsDraft.timezone !== (schedule.timezone ?? ""))
      patch.timezone = detailsDraft.timezone;
    if (detailsDraft.autoDelete !== !!schedule.autoDelete)
      patch.autoDelete = detailsDraft.autoDelete;
    if (detailsDraft.keepAgents !== !!schedule.keepAgents)
      patch.keepAgents = detailsDraft.keepAgents;
    if (detailsDraft.suspended !== !!schedule.suspended)
      patch.suspended = detailsDraft.suspended;
    // Inline agent template — only when no agentName is targeted. The API
    // replaces spec.agent wholesale, so always send the full spec built from
    // the form. Compare against the current spec normalised through the same
    // builder so "30m0s" vs "30m" doesn't register as a change.
    if (!draftAgent) {
      const cur = schedule.agent;
      const next = buildScheduleAgentSpec(detailsDraft.agent);
      const before = cur ? buildScheduleAgentSpec(agentFormValuesFromScheduleSpec(cur, schedule.namespace)) : undefined;
      // Labels have no form input: carry them over, and mirror onto `before` so
      // they never count as a change on their own.
      if (cur?.labels) {
        next.labels = cur.labels;
        if (before) before.labels = cur.labels;
      }
      const switchedToNewAgent = !!currentAgent;
      if (switchedToNewAgent || JSON.stringify(before) !== JSON.stringify(next)) {
        patch.agent = next;
      }
    }
    if (Object.keys(patch).length === 0) {
      setEditingDetails(false);
      return;
    }
    setSavingDetails(true);
    try {
      await patchSchedule(name, patch, scheduleNs);
      await fetchData();
      setEditingDetails(false);
    } catch {
      // keep editing open on error
    } finally {
      setSavingDetails(false);
    }
  }

  async function handleToggleSuspended() {
    if (!schedule) return;
    try {
      await patchSchedule(name, { suspended: !schedule.suspended }, scheduleNs);
      await fetchData();
    } catch {
      /* non-critical */
    }
  }

  async function handleDelete() {
    try {
      await deleteSchedule(name, scheduleNs);
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
      await patchSchedule(name, { schedule: trimmed }, scheduleNs);
      await fetchData();
      setEditingCron(false);
    } catch {
      // keep editing on error
    } finally {
      setSavingCron(false);
    }
  }

  async function handleSaveInstructions() {
    const trimmed = instructionsDraft.trim();
    if (!trimmed || trimmed === schedule?.instructions) {
      setEditingInstructions(false);
      return;
    }
    setSavingInstructions(true);
    try {
      await patchSchedule(name, { instructions: trimmed }, scheduleNs);
      await fetchData();
      setEditingInstructions(false);
    } catch {
      // keep editing on error
    } finally {
      setSavingInstructions(false);
    }
  }

  async function handleTrigger() {
    setTriggering(true);
    setTriggerMessage(null);
    try {
      await triggerSchedule(name, scheduleNs);
      setTriggerMessage("Schedule triggered.");
      await fetchData();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Failed to trigger";
      setTriggerMessage(msg);
    } finally {
      setTriggering(false);
      window.setTimeout(() => setTriggerMessage(null), 4000);
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

  // Inline agent template — present only when the schedule creates a new agent
  // each run rather than referencing an existing one.
  const agentSpec = schedule.agent;
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
          <div className="ml-auto flex items-center gap-2">
            {triggerMessage && (
              <span className="text-xs text-[var(--color-text-secondary)]">{triggerMessage}</span>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={handleToggleSuspended}
              title={schedule.suspended ? "Resume scheduled runs" : "Pause without deleting"}
            >
              {schedule.suspended ? (
                <Play className="size-3.5 text-emerald-400" />
              ) : (
                <Pause className="size-3.5 text-[var(--color-text-secondary)]" />
              )}
              {schedule.suspended ? "Resume" : "Suspend"}
            </Button>
            <ConfirmDialog
              title={`Run ${schedule.name} now?`}
              description="This triggers the schedule's instructions immediately, outside of its cron cadence."
              onConfirm={handleTrigger}
              confirmLabel="Run now"
              confirmVariant="primary"
              trigger={
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={triggering || schedule.lastRunStatus === "InProgress"}
                  title={schedule.lastRunStatus === "InProgress" ? "A run is already in progress" : undefined}
                >
                  <Play className="size-3.5 text-[var(--color-brand-blue)]" />
                  {triggering ? "Triggering..." : "Run now"}
                </Button>
              }
            />
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

        {/* Instructions */}
        <section>
          <div className="mb-2 flex items-center justify-between">
            <h2 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
              Instructions
            </h2>
            {!editingInstructions && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setInstructionsDraft(schedule.instructions ?? "");
                  setEditingInstructions(true);
                }}
              >
                <Pencil className="size-3 text-[var(--color-text-muted)]" />
                Edit
              </Button>
            )}
          </div>
          {editingInstructions ? (
            <div className="space-y-2">
              <textarea
                value={instructionsDraft}
                onChange={(e) => setInstructionsDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Escape") setEditingInstructions(false);
                }}
                className="w-full min-h-[120px] rounded-md border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm text-[var(--color-text)] outline-none focus:border-[var(--color-brand-blue)] resize-y"
                placeholder="What should the agent do each run?"
                autoFocus
                disabled={savingInstructions}
              />
              <div className="flex items-center gap-2">
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleSaveInstructions}
                  disabled={savingInstructions}
                >
                  <Check className="size-3.5" />
                  {savingInstructions ? "Saving..." : "Save"}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setEditingInstructions(false)}
                  disabled={savingInstructions}
                >
                  <X className="size-3.5" />
                  Cancel
                </Button>
              </div>
            </div>
          ) : (
            <div className="rounded-md border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm text-[var(--color-text)] whitespace-pre-wrap">
              {schedule.instructions || (
                <span className="text-[var(--color-text-muted)] italic">No instructions set.</span>
              )}
            </div>
          )}
        </section>

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

        {/* Info / Details section — view + inline editor */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
              Details
            </h2>
            {!editingDetails ? (
              <Button variant="ghost" size="sm" onClick={openDetailsEditor}>
                <Pencil className="size-3 text-[var(--color-text-muted)]" />
                Edit
              </Button>
            ) : (
              <div className="flex items-center gap-2">
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleSaveDetails}
                  disabled={savingDetails}
                >
                  <Save className="size-3.5" />
                  {savingDetails ? "Saving..." : "Save"}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setEditingDetails(false)}
                  disabled={savingDetails}
                >
                  <X className="size-3.5" />
                  Cancel
                </Button>
              </div>
            )}
          </div>

          {!editingDetails || !detailsDraft ? (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              {/* Agent */}
              <div>
                <span className="text-xs text-[var(--color-text-secondary)]">Agent</span>
                <p className="mt-0.5">
                  {schedule.agentName ? (
                    <Link
                      href={`/agents/${schedule.agentName}?namespace=${schedule.namespace}`}
                      className="text-sm font-medium text-[var(--color-brand-blue)] hover:underline"
                    >
                      {schedule.agentName}
                    </Link>
                  ) : schedule.agent ? (
                    <span className="text-sm text-[var(--color-text)]">
                      New agent · {schedule.agent.model || "default model"}
                      {schedule.agent.lifecycle ? ` · ${schedule.agent.lifecycle}` : ""}
                    </span>
                  ) : (
                    <span className="text-sm text-[var(--color-text-secondary)]">--</span>
                  )}
                </p>
              </div>

              {/* Timezone */}
              <div>
                <span className="text-xs text-[var(--color-text-secondary)]">Timezone</span>
                <p className="mt-0.5 text-sm text-[var(--color-text)]">
                  {schedule.timezone || "UTC"}
                </p>
              </div>

              {/* Next run */}
              <div>
                <span className="text-xs text-[var(--color-text-secondary)]">Next Run</span>
                <p className="mt-0.5">
                  {schedule.nextRunTime ? (
                    <RelativeTime timestamp={schedule.nextRunTime} />
                  ) : (
                    <span className="text-sm text-[var(--color-text-secondary)]">--</span>
                  )}
                </p>
              </div>

              {/* Last run */}
              <div>
                <span className="text-xs text-[var(--color-text-secondary)]">Last Run</span>
                <div className="mt-0.5 flex items-center gap-2">
                  {schedule.lastRunTime ? (
                    <>
                      <RelativeTime timestamp={schedule.lastRunTime} />
                      {schedule.lastRunStatus && (
                        <StatusBadge status={schedule.lastRunStatus} size="sm" />
                      )}
                    </>
                  ) : (
                    <span className="text-sm text-[var(--color-text-secondary)]">--</span>
                  )}
                </div>
              </div>

              {/* Flags */}
              <div className="sm:col-span-2">
                <span className="text-xs text-[var(--color-text-secondary)]">Flags</span>
                <div className="mt-0.5 flex items-center gap-2">
                  {schedule.suspended ? (
                    <Badge variant="secondary" className="text-[10px]">
                      Suspended
                    </Badge>
                  ) : null}
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
                  {!schedule.suspended && !schedule.autoDelete && !schedule.keepAgents && (
                    <span className="text-sm text-[var(--color-text-secondary)]">--</span>
                  )}
                </div>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              {/* Target agent */}
              <div className="flex flex-col gap-1.5 sm:col-span-2">
                <Label>Target Agent</Label>
                <Select
                  value={detailsDraft.agentName || "__new__"}
                  onValueChange={(v) =>
                    setDetailsDraft({
                      ...detailsDraft,
                      agentName: v === "__new__" ? "" : v,
                    })
                  }
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__new__">Create a new agent each run</SelectItem>
                    {availableAgents.map((a) => (
                      <SelectItem key={a.name} value={a.name}>
                        {a.name}
                        {a.namespace && a.namespace !== schedule.namespace ? ` (${a.namespace})` : ""}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-[11px] text-[var(--color-text-muted)]">
                  Reference an existing agent, or leave on &quot;Create a new agent each run&quot; to use the template below.
                </p>
              </div>

              {/* Timezone */}
              <div className="flex flex-col gap-1.5">
                <Label>Timezone</Label>
                <input
                  value={detailsDraft.timezone}
                  onChange={(e) =>
                    setDetailsDraft({ ...detailsDraft, timezone: e.target.value })
                  }
                  placeholder="UTC"
                  className="h-9 rounded-md border border-[var(--color-border)] bg-[var(--color-surface)] px-3 text-sm text-[var(--color-text)] outline-none focus:border-[var(--color-brand-blue)]"
                />
              </div>

              {/* Flags */}
              <div className="flex flex-col gap-1.5">
                <Label>Flags</Label>
                <div className="flex flex-wrap items-center gap-2">
                  <button
                    type="button"
                    onClick={() =>
                      setDetailsDraft({ ...detailsDraft, suspended: !detailsDraft.suspended })
                    }
                    className={`text-xs px-2.5 py-1 rounded-full border transition-colors cursor-pointer ${
                      detailsDraft.suspended
                        ? "border-[var(--color-text)] bg-white/10 text-[var(--color-text)]"
                        : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
                    }`}
                  >
                    {detailsDraft.suspended && <Check className="inline size-2.5 mr-1" />}
                    Suspended
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      setDetailsDraft({ ...detailsDraft, autoDelete: !detailsDraft.autoDelete })
                    }
                    className={`text-xs px-2.5 py-1 rounded-full border transition-colors cursor-pointer ${
                      detailsDraft.autoDelete
                        ? "border-[var(--color-text)] bg-white/10 text-[var(--color-text)]"
                        : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
                    }`}
                  >
                    {detailsDraft.autoDelete && <Check className="inline size-2.5 mr-1" />}
                    Delete after first run
                  </button>
                  {detailsDraft.autoDelete && (
                    <button
                      type="button"
                      onClick={() =>
                        setDetailsDraft({ ...detailsDraft, keepAgents: !detailsDraft.keepAgents })
                      }
                      className={`text-xs px-2.5 py-1 rounded-full border transition-colors cursor-pointer ${
                        detailsDraft.keepAgents
                          ? "border-[var(--color-text)] bg-white/10 text-[var(--color-text)]"
                          : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
                      }`}
                    >
                      {detailsDraft.keepAgents && <Check className="inline size-2.5 mr-1" />}
                      Keep agents after deletion
                    </button>
                  )}
                </div>
              </div>

              {/* Inline agent template — only when no agentName is targeted */}
              {!detailsDraft.agentName.trim() && (
                <>
                  <div className="flex flex-col gap-1.5 sm:col-span-2">
                    <div className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)] pt-2 border-t border-[var(--color-border)]">
                      Agent template (used to create a new agent each run)
                    </div>
                  </div>
                  <div className="sm:col-span-2">
                    <AgentFieldsForm
                      values={detailsDraft.agent}
                      onChange={(agent) => setDetailsDraft({ ...detailsDraft, agent })}
                      active={editingDetails}
                      hideNameAndNamespace
                      hideInstructions
                      idPrefix="sched-edit"
                    />
                  </div>
                </>
              )}
            </div>
          )}
        </section>

        {/* Agent configuration — the inline template used to create an agent
            each run. Hidden while the Details editor is open, since that editor
            already surfaces the fields it can change. */}
        {agentSpec && !editingDetails && (
          <section>
            <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
              Agent Configuration
            </h2>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              {(
                [
                  ["Model", agentSpec.model, true],
                  ["Role", agentSpec.role, false],
                  ["Lifecycle", agentSpec.lifecycle, false],
                  ["Template", agentSpec.templateRef, true],
                  ["Priority", agentSpec.priority ? String(agentSpec.priority) : undefined, false],
                  ["Storage", agentSpec.storage?.size, false],
                  ["Sleep TTL", agentSpec.sleepTTL ? fmtDuration(agentSpec.sleepTTL) : undefined, false],
                  ["Delete TTL", agentSpec.deleteTTL ? fmtDuration(agentSpec.deleteTTL) : undefined, false],
                  ["Task Timeout", agentSpec.taskTimeout ? fmtDuration(agentSpec.taskTimeout) : undefined, false],
                ] as const
              )
                .filter(([, value]) => value)
                .map(([label, value, mono]) => (
                  <div key={label}>
                    <span className="text-xs text-[var(--color-text-secondary)]">{label}</span>
                    <p
                      className={`mt-0.5 text-sm text-[var(--color-text)] ${mono ? "font-mono" : ""}`}
                    >
                      {value}
                    </p>
                  </div>
                ))}

              {(
                [
                  ["Skills", agentSpec.skills],
                  ["Memories", agentSpec.memories],
                  ["Connectors", agentSpec.connectors],
                  ["Secrets", agentSpec.secrets],
                  ["Allowed Tools", agentSpec.allowedTools],
                  ["Disallowed Tools", agentSpec.disallowedTools],
                ] as const
              )
                .filter(([, items]) => items?.length)
                .map(([label, items]) => {
                  const blocked = label === "Disallowed Tools";
                  return (
                    <div key={label} className="sm:col-span-2">
                      <span className="text-xs text-[var(--color-text-secondary)]">{label}</span>
                      <div className="mt-1 flex flex-wrap gap-1.5">
                        {items?.map((item) => (
                          <Badge
                            key={item}
                            variant={blocked ? "outline" : "secondary"}
                            className={blocked ? "font-mono line-through" : "font-mono"}
                          >
                            {item}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  );
                })}

              {agentSpec.labels && Object.keys(agentSpec.labels).length > 0 && (
                <div className="sm:col-span-2">
                  <span className="text-xs text-[var(--color-text-secondary)]">Labels</span>
                  <div className="mt-1 flex flex-wrap gap-1.5">
                    {Object.entries(agentSpec.labels).map(([k, v]) => (
                      <Badge key={k} variant="secondary" className="font-mono">{k}={v}</Badge>
                    ))}
                  </div>
                </div>
              )}

              {agentSpec.systemPrompt && (
                <div className="sm:col-span-2">
                  <span className="text-xs text-[var(--color-text-secondary)]">System Prompt</span>
                  <div className="mt-1 rounded-md border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm text-[var(--color-text)] whitespace-pre-wrap">
                    {agentSpec.systemPrompt}
                  </div>
                </div>
              )}
            </div>
          </section>
        )}
      </motion.div>
    </div>
  );
}
