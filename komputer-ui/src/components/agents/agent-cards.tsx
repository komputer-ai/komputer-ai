"use client";

import Link from "next/link";
import { Bot, Trash2, Zap, Moon, Skull, Clock, CheckCircle2 } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { Button } from "@/components/kit/button";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { formatCost, formatRelativeTime } from "@/lib/utils";
import type { AgentResponse } from "@/lib/types";

type AgentCardsProps = {
  agents: AgentResponse[];
  onDelete: (name: string) => void;
};

const statusConfig: Record<string, { color: string; icon: typeof Bot; pulse?: boolean }> = {
  Running:   { color: "#34D399", icon: Zap, pulse: true },
  Sleeping:  { color: "#FBBF24", icon: Moon },
  Pending:   { color: "#FBBF24", icon: Clock },
  Failed:    { color: "#F87171", icon: Skull },
  Succeeded: { color: "#34D399", icon: CheckCircle2 },
};

const defaultStatus = { color: "#8899A6", icon: Bot };

export function AgentCards({ agents, onDelete }: AgentCardsProps) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-2.5">
      <AnimatePresence initial={false}>
        {agents.map((agent, i) => {
          const cfg = statusConfig[agent.status] ?? defaultStatus;
          const StatusIcon = cfg.icon;
          const isActive = agent.taskStatus === "InProgress";

          return (
            <motion.div
              key={agent.name}
              initial={{ opacity: 0, scale: 0.97 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.97 }}
              transition={{ duration: 0.2, delay: i * 0.03 }}
            >
              <Link href={`/agents/${agent.name}?namespace=${agent.namespace}`} className="block group">
                <div className="relative overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] transition-all duration-200 group-hover:border-[var(--color-border-hover)] group-hover:shadow-[0_0_20px_rgba(63,133,217,0.06),0_0_40px_rgba(139,92,246,0.04)]">

                  <div className="p-3 flex flex-col h-full">
                    {/* Top row: icon + name + delete + dot */}
                    <div className="flex items-center gap-2">
                      <div
                        className="flex items-center justify-center w-7 h-7 rounded-md shrink-0"
                        style={{ backgroundColor: `${cfg.color}15` }}
                      >
                        <StatusIcon className="w-3.5 h-3.5" style={{ color: cfg.color }} />
                      </div>
                      <span className="text-[13px] font-semibold text-[var(--color-text)] truncate leading-tight flex-1 min-w-0">
                        {agent.name}
                      </span>
                      <div className="flex items-center gap-1.5 shrink-0">
                        <div onClick={(e) => e.stopPropagation()} className="opacity-0 group-hover:opacity-100 transition-opacity">
                          <ConfirmDialog
                            title={`Delete ${agent.name}?`}
                            description="This will permanently delete this agent and its workspace."
                            onConfirm={() => onDelete(agent.name)}
                            trigger={
                              <Button variant="ghost" size="icon" className="h-5 w-5 p-0">
                                <Trash2 className="w-2.5 h-2.5 text-[var(--color-text-secondary)] hover:text-red-400 transition-colors" />
                              </Button>
                            }
                          />
                        </div>
                        <span
                          className="block w-2 h-2 rounded-full"
                          style={{
                            backgroundColor: cfg.color,
                            boxShadow: (cfg as typeof defaultStatus & { pulse?: boolean }).pulse && isActive
                              ? `0 0 6px ${cfg.color}80`
                              : undefined,
                          }}
                        />
                      </div>
                    </div>

                    {/* Bottom: fields */}
                    <div className="mt-4 space-y-1">
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">status</span>
                        <span className="text-[11px] font-medium text-[var(--color-text-secondary)]">
                          {agent.status}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">model</span>
                        <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text-secondary)] truncate ml-2">
                          {agent.model?.replace("claude-", "")}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">cost</span>
                        <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text)]">
                          {formatCost(agent.totalCostUSD)}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">age</span>
                        <span className="text-[11px] text-[var(--color-text-secondary)]">
                          {formatRelativeTime(agent.createdAt)}
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
