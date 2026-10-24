"use client";

import { Button } from "@/components/kit/button";
import { cn } from "@/lib/utils";

type StatusFilterProps<T extends string> = {
  options: readonly T[];
  value: T;
  onChange: (value: T) => void;
};

/** Status pills for list pages. The selected pill stays readable on hover. */
export function StatusFilter<T extends string>({ options, value, onChange }: StatusFilterProps<T>) {
  return (
    <div className="flex gap-1">
      {options.map((f) => (
        <Button
          key={f}
          size="sm"
          variant="ghost"
          aria-pressed={value === f}
          onClick={() => onChange(f)}
          className={cn(
            "text-xs",
            value === f
              ? "bg-[var(--color-brand-blue)]/15 text-[var(--color-brand-blue)] hover:bg-[var(--color-brand-blue)]/25 hover:text-[var(--color-brand-blue-light)]"
              : "text-[var(--color-text-secondary)]",
          )}
        >
          {f}
        </Button>
      ))}
    </div>
  );
}
