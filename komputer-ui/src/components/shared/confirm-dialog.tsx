"use client";

import { useState, type ReactNode } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/kit/dialog";
import { Button } from "@/components/kit/button";

type ConfirmDialogProps = {
  title: string;
  description: string;
  onConfirm: () => void;
  trigger: ReactNode;
  /** Label for the confirm button. Defaults to "Delete". */
  confirmLabel?: string;
  /** Visual style of the confirm button. Defaults to "destructive". */
  confirmVariant?: "destructive" | "primary" | "secondary";
  /** Extra classes for the confirm button, e.g. a status color. */
  confirmClassName?: string;
};

export function ConfirmDialog({
  title,
  description,
  onConfirm,
  trigger,
  confirmLabel = "Delete",
  confirmVariant = "destructive",
  confirmClassName,
}: ConfirmDialogProps) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <span className="inline-flex" onClick={() => setOpen(true)}>
        {trigger}
      </span>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-md mx-auto bg-[var(--color-surface)] border-[var(--color-border)] text-[var(--color-text)]">
          <DialogHeader>
            <DialogTitle className="text-[var(--color-text)]">{title}</DialogTitle>
            <DialogDescription className="text-[var(--color-text-secondary)]">
              {description}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="mt-4">
            <Button
              variant="secondary"
              className="border-[var(--color-border)] text-[var(--color-text-secondary)]"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button
              variant={confirmVariant}
              className={confirmClassName}
              onClick={() => {
                onConfirm();
                setOpen(false);
              }}
            >
              {confirmLabel}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
