"use client";

import { Check } from "lucide-react";
import { cn } from "@/lib/utils";

interface CheckboxProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
  id?: string;
  className?: string;
  "aria-label"?: string;
}

// The app's checkbox: a small rounded box that fills with the brand blue and
// pops a white check, matching the selection boxes in the agents table.
// A <label htmlFor={id}> next to it toggles it too.
export function Checkbox({ checked, onCheckedChange, disabled, id, className, ...rest }: CheckboxProps) {
  return (
    <button
      type="button"
      role="checkbox"
      id={id}
      aria-checked={checked}
      aria-label={rest["aria-label"]}
      disabled={disabled}
      onClick={() => onCheckedChange(!checked)}
      className={cn(
        "flex size-4 shrink-0 items-center justify-center rounded-[4px] border transition-all duration-150 cursor-pointer",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-brand-blue-glow)]",
        "disabled:cursor-not-allowed disabled:opacity-50",
        checked
          ? "border-[var(--color-brand-blue)] bg-[var(--color-brand-blue)]"
          : "border-[var(--color-text-muted)] bg-transparent hover:border-[var(--color-text-secondary)]",
        className,
      )}
    >
      {checked && <Check className="size-3 text-white" strokeWidth={3} style={{ animation: "agent-check-pop 180ms ease-out" }} />}
    </button>
  );
}
