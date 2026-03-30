"use client";

import Link from "next/link";
import { Building2, Trash2, Users, Crown } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { Button } from "@/components/kit/button";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { formatCost } from "@/lib/utils";
import type { OfficeResponse } from "@/lib/types";

type OfficeCardsProps = {
  offices: OfficeResponse[];
  onDelete: (name: string) => void;
};

const phaseConfig: Record<string, { color: string; label: string }> = {
  InProgress: { color: "#34D399", label: "In Progress" },
  Complete:   { color: "#34D399", label: "Complete" },
  Error:      { color: "#F87171", label: "Error" },
};

export function OfficeCards({ offices, onDelete }: OfficeCardsProps) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-2.5">
      <AnimatePresence initial={false}>
        {offices.map((office, i) => {
          const phase = phaseConfig[office.phase] ?? { color: "#8899A6", label: office.phase };

          return (
            <motion.div
              key={office.name}
              initial={{ opacity: 0, scale: 0.97 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.97 }}
              transition={{ duration: 0.2, delay: i * 0.03 }}
            >
              <Link href={`/offices/${office.name}`} className="block group">
                <div className="relative overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] transition-all duration-200 group-hover:border-[var(--color-border-hover)] group-hover:shadow-[0_0_20px_rgba(63,133,217,0.06),0_0_40px_rgba(139,92,246,0.04)]">

                  <div className="p-3 flex flex-col h-full">
                    {/* Top row: icon + name + delete + dot */}
                    <div className="flex items-center gap-2">
                      <div className="flex items-center justify-center w-7 h-7 rounded-md bg-[var(--color-brand-blue)]/10 shrink-0">
                        <Building2 className="w-3.5 h-3.5 text-[var(--color-brand-blue)]" />
                      </div>
                      <span className="text-[13px] font-semibold text-[var(--color-text)] truncate leading-tight flex-1 min-w-0">
                        {office.name}
                      </span>
                      <div className="flex items-center gap-1.5 shrink-0">
                        <div onClick={(e) => e.stopPropagation()} className="opacity-0 group-hover:opacity-100 transition-opacity">
                          <ConfirmDialog
                            title={`Delete ${office.name}?`}
                            description="This will delete the office and all member agents."
                            onConfirm={() => onDelete(office.name)}
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
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">phase</span>
                        <span className="text-[11px] text-[var(--color-text-secondary)]">{phase.label}</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">manager</span>
                        <span className="text-[11px] text-[var(--color-brand-blue)] truncate ml-2">
                          {office.manager}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">agents</span>
                        <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text-secondary)]">
                          {office.totalAgents} ({office.activeAgents} active)
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">cost</span>
                        <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text)]">
                          {formatCost(office.totalCostUSD)}
                        </span>
                      </div>
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
