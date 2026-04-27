"use client";

import { useState } from "react";
import { LogOut } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/kit/dialog";
import { Button } from "@/components/kit/button";
import { removeSquadMember } from "@/lib/api";
import type { Squad } from "@/lib/types";

type LeaveSquadButtonProps = {
  agentName: string;
  agentNamespace: string;
  squad: Pick<Squad, "name" | "namespace">;
  onSuccess?: () => void;
};

export function LeaveSquadButton({
  agentName,
  agentNamespace,
  squad,
  onSuccess,
}: LeaveSquadButtonProps) {
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleConfirm() {
    setSubmitting(true);
    setError(null);
    try {
      await removeSquadMember(squad.name, squad.namespace, agentName);
      onSuccess?.();
      setOpen(false);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to leave squad");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      <Button variant="secondary" size="sm" onClick={() => setOpen(true)}>
        <LogOut className="size-3" data-icon="inline-start" />
        Leave Squad
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-md mx-auto bg-[var(--color-surface)] border-[var(--color-border)] text-[var(--color-text)] p-6">
          <DialogHeader className="space-y-1.5">
            <DialogTitle className="text-[var(--color-text)]">
              Remove {agentName} from squad {squad.name}?
            </DialogTitle>
            <DialogDescription className="text-[var(--color-text-secondary)]">
              {agentName} will become a solo agent again. Its workspace is preserved.
              Any in-flight task on this member will be cancelled when its container is removed
              from the squad pod.
            </DialogDescription>
          </DialogHeader>

          {agentNamespace !== squad.namespace && (
            <p className="mt-2 text-xs text-[var(--color-text-muted)]">
              ns: {agentNamespace} → squad ns: {squad.namespace}
            </p>
          )}

          {error && (
            <p className="mt-3 text-sm text-red-400">{error}</p>
          )}

          <DialogFooter className="mt-6 gap-2">
            <Button
              variant="secondary"
              onClick={() => setOpen(false)}
              disabled={submitting}
              className="border-[var(--color-border)] text-[var(--color-text-secondary)]"
            >
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleConfirm} disabled={submitting}>
              {submitting ? "Leaving…" : "Leave squad"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
