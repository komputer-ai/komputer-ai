"use client";

import Link from "next/link";
import { Users, Trash2 } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { Button } from "@/components/kit/button";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import type { Squad } from "@/lib/types";

type SquadCardsProps = {
  squads: Squad[];
  onDelete: (name: string, namespace: string) => void;
};

const phaseConfig: Record<string, { color: string; label: string }> = {
  Pending:  { color: "#FBBF24", label: "Pending" },
  Running:  { color: "#34D399", label: "Running" },
  Orphaned: { color: "#8899A6", label: "Orphaned" },
  Failed:   { color: "#F87171", label: "Failed" },
};

export function SquadCards({ squads, onDelete }: SquadCardsProps) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-2.5">
      <AnimatePresence>
        {squads.map((squad, i) => {
          const phase = phaseConfig[squad.phase] ?? { color: "#8899A6", label: squad.phase };
          const readyCount = squad.members.filter((m) => m.ready).length;

          return (
            <motion.div
              key={`${squad.namespace}/${squad.name}`}
              initial={{ opacity: 0, y: 12, scale: 0.97 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, scale: 0.97 }}
              transition={{ duration: 0.25, delay: i * 0.04 }}
            >
              <Link href={`/squads/${squad.name}?namespace=${squad.namespace}`} className="block group">
                <div className="relative overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] transition-all duration-200 group-hover:border-[var(--color-border-hover)] group-hover:shadow-[0_0_20px_rgba(63,133,217,0.06),0_0_40px_rgba(139,92,246,0.04)]">
                  <div className="p-3 flex flex-col h-full">
                    {/* Top row: icon + name + delete + phase dot */}
                    <div className="flex items-center gap-2">
                      <div className="flex items-center justify-center w-7 h-7 rounded-md bg-[var(--color-brand-blue)]/10 shrink-0">
                        <Users className="w-3.5 h-3.5 text-[var(--color-brand-blue)]" />
                      </div>
                      <span className="text-[13px] font-semibold text-[var(--color-text)] truncate leading-tight flex-1 min-w-0">
                        {squad.name}
                      </span>
                      <div className="flex items-center gap-1.5 shrink-0">
                        <div onClick={(e) => { e.stopPropagation(); e.preventDefault(); }} className="opacity-0 group-hover:opacity-100 transition-opacity">
                          <ConfirmDialog
                            title={`Delete ${squad.name}?`}
                            description="This will delete the squad and detach all member agents."
                            onConfirm={() => onDelete(squad.name, squad.namespace)}
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

                    {/* Fields */}
                    <div className="mt-4 space-y-1">
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">namespace</span>
                        <span className="text-[11px] text-[var(--color-text-secondary)] truncate ml-2">{squad.namespace}</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">phase</span>
                        <span className="text-[11px] text-[var(--color-text-secondary)]">{phase.label}</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">members</span>
                        <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text-secondary)]">
                          {squad.members.length} ({readyCount} ready)
                        </span>
                      </div>
                      {squad.members.length > 0 && (
                        <div className="flex flex-wrap gap-1 mt-1.5">
                          {squad.members.slice(0, 3).map((m) => (
                            <span
                              key={m.name}
                              onClick={(e) => { e.stopPropagation(); e.preventDefault(); window.location.href = `/agents/${m.name}`; }}
                              className="text-[10px] px-1.5 py-0.5 rounded bg-[var(--color-surface-raised)] text-[var(--color-brand-blue)] hover:underline cursor-pointer truncate max-w-[80px]"
                            >
                              {m.name}
                            </span>
                          ))}
                          {squad.members.length > 3 && (
                            <span className="text-[10px] px-1.5 py-0.5 rounded bg-[var(--color-surface-raised)] text-[var(--color-text-muted)]">
                              +{squad.members.length - 3}
                            </span>
                          )}
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
