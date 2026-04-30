"use client";

import { useState } from "react";
import { ChevronDown, Plus, Bot } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import type { AgentResponse } from "@/lib/types";
import { RelativeTime } from "@/components/shared/relative-time";

export interface ActiveAgentChipProps {
  active: AgentResponse | null;
  agents: AgentResponse[];
  onSelect: (agent: AgentResponse) => void;
  onNew: () => void;
}

export function ActiveAgentChip({ active, agents, onSelect, onNew }: ActiveAgentChipProps) {
  const [open, setOpen] = useState(false);
  if (!active) {
    return (
      <button
        type="button"
        onClick={onNew}
        className="inline-flex items-center gap-1.5 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors cursor-pointer"
      >
        <Bot className="size-3" /> No personal agent yet — Go will create one
      </button>
    );
  }
  return (
    <div className="relative inline-block">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="inline-flex items-center gap-1.5 rounded-full border border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 py-1 text-xs text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)] hover:text-[var(--color-text)] transition-colors cursor-pointer"
      >
        <Bot className="size-3 text-[var(--color-brand-blue)]" />
        Using <span className="font-medium text-[var(--color-text)]">{active.name}</span>
        <ChevronDown className={`size-3 transition-transform ${open ? "rotate-180" : ""}`} />
      </button>
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.12 }}
            className="absolute z-50 mt-1 min-w-56 rounded-md border border-[var(--color-border)] bg-[var(--color-surface-raised)] shadow-[0_8px_32px_rgba(0,0,0,0.4)]"
          >
            {agents.map((a) => (
              <button
                key={`${a.namespace}/${a.name}`}
                type="button"
                onClick={() => {
                  setOpen(false);
                  onSelect(a);
                }}
                className={`flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm transition-colors hover:bg-[var(--color-surface-hover)] ${
                  a.name === active.name ? "text-[var(--color-brand-blue)]" : "text-[var(--color-text)]"
                }`}
              >
                <span className="truncate">{a.name}</span>
                {a.completionTime ? (
                  <span className="text-[10px] text-[var(--color-text-muted)]">
                    <RelativeTime timestamp={a.completionTime} />
                  </span>
                ) : null}
              </button>
            ))}
            <button
              type="button"
              onClick={() => {
                setOpen(false);
                onNew();
              }}
              className="flex w-full items-center gap-2 border-t border-[var(--color-border)] px-3 py-2 text-left text-xs text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-hover)] hover:text-[var(--color-text)] cursor-pointer"
            >
              <Plus className="size-3" /> New personal agent
            </button>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
