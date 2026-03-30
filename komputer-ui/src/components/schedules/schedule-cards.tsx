"use client";

import Link from "next/link";
import { Clock, Trash2, Play, Pause, AlertTriangle, Repeat, Zap } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { Button } from "@/components/kit/button";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { formatCost, formatRelativeTime, cronToHuman } from "@/lib/utils";
import type { ScheduleResponse } from "@/lib/types";

type ScheduleCardsProps = {
  schedules: ScheduleResponse[];
  onDelete: (name: string) => void;
};

const phaseConfig: Record<string, { color: string; icon: typeof Clock }> = {
  Active:    { color: "#34D399", icon: Play },
  Suspended: { color: "#FBBF24", icon: Pause },
  Error:     { color: "#F87171", icon: AlertTriangle },
};

export function ScheduleCards({ schedules, onDelete }: ScheduleCardsProps) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-2.5">
      <AnimatePresence initial={false}>
        {schedules.map((schedule, i) => {
          const phase = phaseConfig[schedule.phase] ?? { color: "#8899A6", icon: Clock };
          const PhaseIcon = phase.icon;
          const total = schedule.runCount ?? 0;
          const failed = schedule.failedRuns ?? 0;

          return (
            <motion.div
              key={schedule.name}
              initial={{ opacity: 0, scale: 0.97 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.97 }}
              transition={{ duration: 0.2, delay: i * 0.03 }}
            >
              <Link href={`/schedules/${schedule.name}`} className="block group">
                <div className="relative overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] transition-all duration-200 group-hover:border-[var(--color-border-hover)] group-hover:shadow-[0_0_20px_rgba(63,133,217,0.06),0_0_40px_rgba(139,92,246,0.04)]">

                  <div className="p-3 flex flex-col h-full">
                    {/* Top row: icon + name + delete + dot */}
                    <div className="flex items-center gap-2">
                      <div
                        className="flex items-center justify-center w-7 h-7 rounded-md shrink-0"
                        style={{ backgroundColor: `${phase.color}15` }}
                      >
                        <PhaseIcon className="w-3.5 h-3.5" style={{ color: phase.color }} />
                      </div>
                      <span className="text-[13px] font-semibold text-[var(--color-text)] truncate leading-tight flex-1 min-w-0">
                        {schedule.name}
                      </span>
                      <div className="flex items-center gap-1.5 shrink-0">
                        <div onClick={(e) => e.stopPropagation()} className="opacity-0 group-hover:opacity-100 transition-opacity">
                          <ConfirmDialog
                            title={`Delete ${schedule.name}?`}
                            description="This will delete the schedule and managed agents."
                            onConfirm={() => onDelete(schedule.name)}
                            trigger={
                              <Button variant="ghost" size="icon" className="h-5 w-5 p-0">
                                <Trash2 className="w-2.5 h-2.5 text-[var(--color-text-secondary)] hover:text-red-400 transition-colors" />
                              </Button>
                            }
                          />
                        </div>
                        <span className="block w-2 h-2 rounded-full" style={{ backgroundColor: phase.color }} />
                      </div>
                    </div>

                    {/* Bottom: fields */}
                    <div className="mt-4 space-y-1">
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">cron</span>
                        <code className="text-[10px] font-[family-name:var(--font-mono)] text-[var(--color-text-secondary)]">
                          {cronToHuman(schedule.schedule)}
                        </code>
                      </div>
                      {schedule.agentName && (
                        <div className="flex items-center justify-between">
                          <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">agent</span>
                          <span className="text-[11px] text-[var(--color-brand-blue)] truncate ml-2">{schedule.agentName}</span>
                        </div>
                      )}
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">runs</span>
                        <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text-secondary)]">
                          {total}{failed > 0 && <span className="text-red-400 ml-1">({failed}✗)</span>}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">cost</span>
                        <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text)]">
                          {formatCost(schedule.totalCostUSD)}
                        </span>
                      </div>
                      {schedule.nextRunTime && (
                        <div className="flex items-center justify-between">
                          <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">next</span>
                          <span className="text-[11px] text-[var(--color-text-secondary)]">{formatRelativeTime(schedule.nextRunTime)}</span>
                        </div>
                      )}
                    </div>
                  </div>

                </div>
              </Link>
            </motion.div>
          );
        })}
      </AnimatePresence>
    </div>
  );
}
