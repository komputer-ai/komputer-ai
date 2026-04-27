"use client";

import { useState } from "react";
import { Unlink, Hourglass } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/kit/dialog";
import { Button } from "@/components/kit/button";
import { breakUpSquad } from "@/lib/api";
import type { Squad } from "@/lib/types";

type BreakUpButtonProps = {
  squad: Pick<Squad, "name" | "namespace" | "breakUpRequested">;
  onSuccess?: () => void;
  // Visual variant: "secondary" matches the team-up button on the agent page,
  // "primary" is for the squad detail page header.
  variant?: "secondary" | "destructive";
  size?: "sm" | "md";
};

export function BreakUpButton({
  squad,
  onSuccess,
  variant = "secondary",
  size = "sm",
}: BreakUpButtonProps) {
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const alreadyRequested = !!squad.breakUpRequested;

  async function handleConfirm() {
    setSubmitting(true);
    setError(null);
    try {
      await breakUpSquad(squad.name, squad.namespace);
      onSuccess?.();
      setOpen(false);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to request break-up");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      <Button
        variant={variant}
        size={size}
        onClick={() => setOpen(true)}
        disabled={alreadyRequested}
        title={alreadyRequested ? "Break-up already requested" : undefined}
      >
        {alreadyRequested ? (
          <Hourglass className="size-3" data-icon="inline-start" />
        ) : (
          <Unlink className="size-3" data-icon="inline-start" />
        )}
        {alreadyRequested ? "Break-up pending" : "Break Up"}
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-md mx-auto bg-[var(--color-surface)] border-[var(--color-border)] text-[var(--color-text)] p-6">
          <DialogHeader className="space-y-1.5">
            <DialogTitle className="text-[var(--color-text)]">
              Break up squad {squad.name}?
            </DialogTitle>
            <DialogDescription className="text-[var(--color-text-secondary)]">
              The squad will be dissolved once <span className="font-medium text-[var(--color-text)]">all members are sleeping</span>.
              Members will become solo agents — workspaces are preserved.
            </DialogDescription>
          </DialogHeader>

          <div className="mt-4 rounded-[var(--radius-md)] border border-amber-500/40 bg-amber-500/5 p-3 text-sm text-amber-300/90 space-y-1.5">
            <p className="font-medium text-amber-200">
              Heads up: every member must be asleep at the same time for the break-up to take effect.
            </p>
            <p>
              You can sleep members yourself (or use &quot;Sleep all&quot; on the squad page), or wait for each member to finish its current task and sleep on its own.
            </p>
          </div>

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
              {submitting ? "Requesting…" : "Request break-up"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
