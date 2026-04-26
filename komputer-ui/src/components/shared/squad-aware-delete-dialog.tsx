"use client";

import { useState, type ReactNode } from "react";
import Link from "next/link";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/kit/dialog";
import { Button } from "@/components/kit/button";

export type SquadAwareDeleteOptions = { recreatePod?: boolean };

type SquadInfo = { name: string; namespace: string };

type SquadAwareDeleteDialogProps = {
  // Either provide a `trigger` (uncontrolled) OR `open`+`onOpenChange` (controlled).
  trigger?: ReactNode;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  // The dialog's title, e.g. "Delete agent-1?" or "Delete 3 agents?".
  title: string;
  // Top description rendered above the squad context (plain text or rich content).
  description?: ReactNode;
  // Squad context — when provided, the squad checkboxes are rendered.
  squad?: SquadInfo;
  // Squad members that will be affected (used in bulk dialog to list members).
  squadMembers?: { agentName: string; squad: SquadInfo }[];
  // Called with chosen options. If `deleteSquads` was checked, the caller is
  // expected to delete the squad(s) itself (this dialog does not call deleteSquad).
  onConfirm: (opts: { recreatePod: boolean; deleteSquads: boolean }) => Promise<void> | void;
  confirmLabel?: string;
  submittingLabel?: string;
};

export function SquadAwareDeleteDialog({
  trigger,
  open: openProp,
  onOpenChange,
  title,
  description,
  squad,
  squadMembers,
  onConfirm,
  confirmLabel = "Delete",
  submittingLabel = "Deleting...",
}: SquadAwareDeleteDialogProps) {
  const isControlled = openProp !== undefined;
  const [openState, setOpenState] = useState(false);
  const open = isControlled ? !!openProp : openState;
  const setOpen = (next: boolean) => {
    if (!isControlled) setOpenState(next);
    onOpenChange?.(next);
  };
  const [deleteSquadChecked, setDeleteSquadChecked] = useState(false);
  const [recreatePodChecked, setRecreatePodChecked] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const showSquadOptions = !!squad || (squadMembers && squadMembers.length > 0);

  const handleConfirm = async () => {
    setSubmitting(true);
    try {
      await onConfirm({
        recreatePod: recreatePodChecked && !deleteSquadChecked,
        deleteSquads: deleteSquadChecked,
      });
      setOpen(false);
      setDeleteSquadChecked(false);
      setRecreatePodChecked(false);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      {trigger && (
        <span className="inline-flex" onClick={() => setOpen(true)}>
          {trigger}
        </span>
      )}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-md mx-auto bg-[var(--color-surface)] border-[var(--color-border)] text-[var(--color-text)] p-6">
          <DialogHeader className="space-y-1.5">
            <DialogTitle className="text-[var(--color-text)]">{title}</DialogTitle>
            {description && (
              <DialogDescription className="text-[var(--color-text-secondary)]">
                {description}
              </DialogDescription>
            )}
            {squad && (
              <DialogDescription className="text-[var(--color-text-secondary)]">
                This agent is part of squad{" "}
                <Link
                  href={`/squads/${squad.name}?namespace=${squad.namespace}`}
                  className="font-medium text-[var(--color-brand-blue)] hover:underline"
                  onClick={() => setOpen(false)}
                >
                  {squad.name}
                </Link>
                .
              </DialogDescription>
            )}
          </DialogHeader>

          <div className="mt-5 space-y-4">
            {squadMembers && squadMembers.length > 0 && (
              <div className="rounded-[var(--radius-md)] border border-violet-500/30 bg-violet-500/5 p-3 space-y-2">
                <p className="text-xs uppercase tracking-wider text-violet-300/80">
                  Squad members in selection
                </p>
                <ul className="text-sm text-[var(--color-text-secondary)] space-y-1">
                  {squadMembers.map((m) => (
                    <li key={`${m.squad.namespace}/${m.agentName}`} className="truncate">
                      <span className="text-[var(--color-text)]">{m.agentName}</span>
                      <span className="text-[var(--color-text-muted)]"> · squad </span>
                      <Link
                        href={`/squads/${m.squad.name}?namespace=${m.squad.namespace}`}
                        className="text-violet-300 hover:underline"
                        onClick={() => setOpen(false)}
                      >
                        {m.squad.name}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {showSquadOptions && (
              <div className="space-y-3">
                <label className="flex items-start gap-3 cursor-pointer group">
                  <input
                    type="checkbox"
                    checked={deleteSquadChecked}
                    onChange={(e) => setDeleteSquadChecked(e.target.checked)}
                    className="mt-0.5 accent-[var(--color-brand-blue)]"
                  />
                  <span className="text-sm text-[var(--color-text-secondary)] group-hover:text-[var(--color-text)] transition-colors leading-snug">
                    Delete squad too (other members will revert to solo agents)
                  </span>
                </label>
                <label
                  className={`flex items-start gap-3 group ${
                    deleteSquadChecked ? "opacity-50 cursor-not-allowed" : "cursor-pointer"
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={recreatePodChecked && !deleteSquadChecked}
                    onChange={(e) => setRecreatePodChecked(e.target.checked)}
                    disabled={deleteSquadChecked}
                    className="mt-0.5 accent-[var(--color-brand-blue)]"
                  />
                  <span className="text-sm text-[var(--color-text-secondary)] group-hover:text-[var(--color-text)] transition-colors leading-snug">
                    Recreate squad pod (clears the removed member&apos;s container; brief restart of remaining members)
                  </span>
                </label>
              </div>
            )}
          </div>

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
              {submitting ? submittingLabel : confirmLabel}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
