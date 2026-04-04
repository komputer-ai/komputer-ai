"use client";

import { useState, useCallback } from "react";
import Link from "next/link";
import { KeyRound, Trash2, Users, Plus } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { Button } from "@/components/kit/button";
import { Input } from "@/components/kit/input";
import { Label } from "@/components/kit/label";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/kit/dialog";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { formatRelativeTime } from "@/lib/utils";
import { updateSecretResource } from "@/lib/api";
import type { SecretResponse } from "@/lib/types";

type KeyValueEntry = { key: string; value: string };

type SecretCardsProps = {
  secrets: SecretResponse[];
  onDelete: (name: string, namespace: string) => void;
  onUpdated?: () => void;
};

function EditSecretDialog({ secret, onClose, onUpdated }: { secret: SecretResponse; onClose: () => void; onUpdated?: () => void }) {
  const [pairs, setPairs] = useState<KeyValueEntry[]>(
    secret.keys.length > 0 ? secret.keys.map((k) => ({ key: k, value: "" })) : [{ key: "", value: "" }]
  );
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const addPair = useCallback(() => setPairs((prev) => [...prev, { key: "", value: "" }]), []);
  const removePair = useCallback((i: number) => setPairs((prev) => prev.filter((_, idx) => idx !== i)), []);
  const updatePair = useCallback((i: number, field: "key" | "value", val: string) => {
    setPairs((prev) => prev.map((p, idx) => (idx === i ? { ...p, [field]: val } : p)));
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const filled = pairs.filter((p) => p.key.trim() && p.value.trim());
    if (filled.length === 0) { setError("At least one key-value pair with a value is required."); return; }
    setSubmitting(true);
    setError(null);
    try {
      const data: Record<string, string> = {};
      for (const p of filled) data[p.key.trim()] = p.value.trim();
      await updateSecretResource(secret.name, data, secret.namespace);
      onUpdated?.();
      onClose();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to update secret.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent className="w-full max-w-md">
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <KeyRound className="size-4 text-amber-400" />
              Edit {secret.name}
            </DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-1.5">
            <Label>Key-Value Pairs</Label>
            <p className="text-[11px] text-[var(--color-text-muted)]">Leave value blank to keep existing value for that key.</p>
            <div className="flex flex-col gap-2 mt-1">
              {pairs.map((pair, i) => (
                <div key={i} className="flex items-center gap-2">
                  <Input
                    placeholder="KEY"
                    value={pair.key}
                    onChange={(e) => updatePair(i, "key", e.target.value)}
                    autoComplete="off"
                    className="flex-1"
                  />
                  <Input
                    type="password"
                    placeholder="new value"
                    value={pair.value}
                    onChange={(e) => updatePair(i, "value", e.target.value)}
                    autoComplete="off"
                    className="flex-1"
                  />
                  <Button type="button" variant="ghost" size="icon" onClick={() => removePair(i)} disabled={pairs.length === 1} className="shrink-0">
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              ))}
              <Button type="button" variant="secondary" size="sm" onClick={addPair} className="w-fit">
                <Plus className="mr-1 h-4 w-4" />
                Add Key
              </Button>
            </div>
          </div>
          {error && <p className="text-sm text-red-400">{error}</p>}
          <DialogFooter>
            <Button variant="secondary" type="button" onClick={onClose}>Cancel</Button>
            <Button type="submit" disabled={submitting}>{submitting ? "Saving..." : "Save"}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export function SecretCards({ secrets, onDelete, onUpdated }: SecretCardsProps) {
  const [inspecting, setInspecting] = useState<SecretResponse | null>(null);
  const [editing, setEditing] = useState<SecretResponse | null>(null);

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
                onClick={() => secret.managed ? setEditing(secret) : setInspecting(secret)}
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
                    <div className="flex items-center gap-1 shrink-0">
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
                      <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">agents</span>
                      <span className="text-[11px] text-[var(--color-text-secondary)] flex items-center gap-1">
                        <Users className="w-2.5 h-2.5" />
                        {secret.attachedAgents}
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

      {/* Keys inspect dialog */}
      <Dialog open={!!inspecting} onOpenChange={(open) => { if (!open) setInspecting(null); }}>
        <DialogContent className="w-full max-w-sm">
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
              {inspecting.agentNames && inspecting.agentNames.length > 0 && (
                <div className="mt-4">
                  <p className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)] mb-1.5">Used by agents</p>
                  <div className="space-y-1">
                    {inspecting.agentNames.map((name) => (
                      <Link
                        key={name}
                        href={`/agents/${name}?namespace=${inspecting.namespace}`}
                        className="flex items-center gap-2 rounded-[var(--radius-sm)] bg-[var(--color-bg)] px-3 py-2 text-xs text-[var(--color-brand-blue-light)] hover:underline"
                        onClick={() => setInspecting(null)}
                      >
                        {name}
                      </Link>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </DialogContent>
      </Dialog>

      {/* Edit dialog */}
      {editing && (
        <EditSecretDialog
          secret={editing}
          onClose={() => setEditing(null)}
          onUpdated={() => { setEditing(null); onUpdated?.(); }}
        />
      )}
    </>
  );
}
