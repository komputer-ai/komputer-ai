"use client";

import Link from "next/link";
import { Brain, Trash2, Users } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { Button } from "@/components/kit/button";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { formatRelativeTime } from "@/lib/utils";
import type { MemoryResponse } from "@/lib/types";

type MemoryCardsProps = {
  memories: MemoryResponse[];
  onDelete: (name: string, namespace: string) => void;
};

export function MemoryCards({ memories, onDelete }: MemoryCardsProps) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-2.5">
      <AnimatePresence>
        {memories.map((memory, i) => (
          <motion.div
            key={`${memory.namespace}/${memory.name}`}
            initial={{ opacity: 0, y: 12, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, scale: 0.97 }}
            transition={{ duration: 0.25, delay: i * 0.04 }}
            className="h-full"
          >
            <Link href={`/memories/${memory.name}?namespace=${memory.namespace}`} className="block group">
              <div className="relative h-full min-h-32 overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] transition-all duration-200 group-hover:border-[var(--color-border-hover)] group-hover:shadow-[0_0_20px_rgba(139,92,246,0.06)]">
                <div className="flex h-full flex-col p-3">
                  <div className="flex items-center gap-2">
                    <div className="flex items-center justify-center w-7 h-7 rounded-md shrink-0 bg-[var(--color-brand-violet)]/10">
                      <Brain className="w-3.5 h-3.5 text-[var(--color-brand-violet)]" />
                    </div>
                    <span className="text-[13px] font-semibold text-[var(--color-text)] truncate leading-tight flex-1 min-w-0">
                      {memory.name}
                    </span>
                    <div className="flex items-center gap-1.5 shrink-0">
                      <div onClick={(e) => { e.stopPropagation(); e.preventDefault(); }} className="opacity-0 group-hover:opacity-100 transition-opacity">
                        <ConfirmDialog
                          title={`Delete ${memory.name}?`}
                          description="This will permanently delete this memory."
                          onConfirm={() => onDelete(memory.name, memory.namespace)}
                          trigger={
                            <Button variant="ghost" size="icon" className="h-5 w-5 p-0">
                              <Trash2 className="w-2.5 h-2.5 text-[var(--color-text-secondary)] hover:text-red-400 transition-colors" />
                            </Button>
                          }
                        />
                      </div>
                    </div>
                  </div>

                  <div className="mt-2 min-h-[2.75rem]">
                    {memory.description && (
                      <p className="text-[11px] text-[var(--color-text-secondary)] line-clamp-2">
                        {memory.description}
                      </p>
                    )}
                  </div>

                  <div className="mt-auto pt-3 space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">agents</span>
                      <span className="text-[11px] text-[var(--color-text-secondary)] flex items-center gap-1">
                        <Users className="w-2.5 h-2.5" />
                        {memory.attachedAgents}
                      </span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">age</span>
                      <span className="text-[11px] text-[var(--color-text-secondary)]">
                        {formatRelativeTime(memory.createdAt)}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </Link>
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  );
}
