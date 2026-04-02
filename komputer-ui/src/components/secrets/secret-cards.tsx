"use client";

import { useState } from "react";
import Link from "next/link";
import { KeyRound, Trash2 } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { Button } from "@/components/kit/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/kit/dialog";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { formatRelativeTime } from "@/lib/utils";
import type { SecretResponse } from "@/lib/types";

type SecretCardsProps = {
  secrets: SecretResponse[];
  onDelete: (name: string, namespace: string) => void;
};

export function SecretCards({ secrets, onDelete }: SecretCardsProps) {
  const [inspecting, setInspecting] = useState<SecretResponse | null>(null);

  return (
    <>
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-2.5">
        <AnimatePresence>
          {secrets.map((secret, i) => (
            <motion.div
              key={`${secret.namespace}/${secret.name}`}
              initial={{ opacity: 0, y: 12, scale: 0.97 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, scale: 0.97 }}
              transition={{ duration: 0.25, delay: i * 0.04 }}
              className="h-full"
            >
              <div
                className="group relative h-full min-h-32 overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] transition-all duration-200 hover:border-[var(--color-border-hover)] hover:shadow-[0_0_20px_rgba(245,158,11,0.06)] cursor-pointer"
                onClick={() => setInspecting(secret)}
              >
                <div className="flex h-full flex-col p-3">
                  <div className="flex items-center gap-2">
                    <div className="flex items-center justify-center w-7 h-7 rounded-md shrink-0 bg-amber-500/10">
                      <KeyRound className="w-3.5 h-3.5 text-amber-400" />
                    </div>
                    <span className="text-[13px] font-semibold text-[var(--color-text)] truncate leading-tight flex-1 min-w-0">
                      {secret.name}
                    </span>
                    {!secret.managed && (
                      <span className="inline-flex items-center text-[9px] tracking-wider px-1.5 py-0.5 rounded bg-[var(--color-surface-hover)] text-[var(--color-text-muted)] shrink-0 leading-none">
                        external
                      </span>
                    )}
                    <div className="flex items-center gap-1.5 shrink-0">
                      <div onClick={(e) => e.stopPropagation()} className="opacity-0 group-hover:opacity-100 transition-opacity">
                        <ConfirmDialog
                          title={`Delete ${secret.name}?`}
                          description="This will permanently delete this secret."
                          onConfirm={() => onDelete(secret.name, secret.namespace)}
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
                    <p className="text-[11px] text-[var(--color-text-secondary)]">
                      {secret.keys.length} {secret.keys.length === 1 ? "key" : "keys"}
                    </p>
                    {secret.agentName && (
                      <Link
                        href={`/agents/${secret.agentName}`}
                        className="text-[11px] text-[var(--color-brand-blue-light)] hover:underline truncate block"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {secret.agentName}
                      </Link>
                    )}
                  </div>

                  <div className="mt-auto pt-3 space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">namespace</span>
                      <span className="text-[11px] text-[var(--color-text-secondary)] truncate max-w-[60%] text-right">
                        {secret.namespace}
                      </span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">age</span>
                      <span className="text-[11px] text-[var(--color-text-secondary)]">
                        {formatRelativeTime(secret.createdAt)}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </motion.div>
          ))}
        </AnimatePresence>
      </div>

      {/* Keys dialog */}
      <Dialog open={!!inspecting} onOpenChange={(open) => { if (!open) setInspecting(null); }}>
        <DialogContent className="max-w-sm">
          {inspecting && (
            <>
              <DialogHeader>
                <DialogTitle className="flex items-center gap-2">
                  <KeyRound className="size-4 text-amber-400" />
                  {inspecting.name}
                </DialogTitle>
              </DialogHeader>
              <div className="mt-3 space-y-1.5">
                {inspecting.keys.map((key) => (
                  <div key={key} className="flex items-center gap-2 rounded-[var(--radius-sm)] bg-[var(--color-bg)] px-3 py-2">
                    <span className="text-xs font-[family-name:var(--font-mono)] text-[var(--color-text-secondary)] truncate">
                      {key}
                    </span>
                    <span className="ml-auto text-[10px] tracking-widest text-[var(--color-text-muted)]">
                      ••••••
                    </span>
                  </div>
                ))}
                {inspecting.keys.length === 0 && (
                  <p className="text-xs text-[var(--color-text-muted)]">No keys</p>
                )}
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
