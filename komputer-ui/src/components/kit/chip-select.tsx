"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { Check, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";

export interface ChipSelectOption {
  /** Stable identifier — passed to onChange when selected. */
  value: string;
  /** Primary label rendered in the dropdown. */
  label: ReactNode;
  /** Optional icon at the start of the row. */
  icon?: ReactNode;
  /** Optional secondary text aligned to the right of the row (e.g. timestamp). */
  meta?: ReactNode;
  /** Optional className override for the row text color when not selected. */
  className?: string;
}

export interface ChipSelectProps {
  /** Currently selected value. Used to style the row + the trigger label. */
  value: string;
  /** Options shown in the dropdown body. */
  options: ChipSelectOption[];
  /** Called when the user picks an option. */
  onChange: (value: string) => void;
  /** Trigger contents — typically an icon + label. */
  trigger: ReactNode;
  /** Optional content rendered below the option list (e.g. an "Add" form).
   *
   * Pass a `ReactNode` for static content, or a render-prop that receives
   * an imperative `close` so the footer can dismiss the dropdown after
   * committing a value (e.g. after confirming an inline "Add" form). */
  footer?: ReactNode | ((api: { close: () => void }) => ReactNode);
  /** Optional className for the trigger button. */
  triggerClassName?: string;
}

/**
 * A pill-style dropdown selector. Single trigger button rounded-full, opens a
 * dropdown of options, optional footer slot for inline add/edit forms.
 *
 * Used by ActiveAgentChip and NamespaceChip on the dashboard prompt bar.
 */
export function ChipSelect({
  value,
  options,
  onChange,
  trigger,
  footer,
  triggerClassName,
}: ChipSelectProps) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  return (
    <div className="relative inline-block" ref={containerRef}>
      <motion.button
        type="button"
        onClick={() => setOpen((o) => !o)}
        layout
        transition={{ type: "spring", stiffness: 500, damping: 35, mass: 0.5 }}
        className={cn(
          "inline-flex items-center gap-1.5 rounded-full border border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 py-1 text-xs text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)] hover:text-[var(--color-text)] transition-colors cursor-pointer",
          triggerClassName,
        )}
      >
        <AnimatePresence mode="wait" initial={false}>
          <motion.span
            key={value}
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 4 }}
            transition={{ duration: 0.16, ease: "easeOut" }}
            className="inline-flex items-center gap-1.5"
          >
            {trigger}
          </motion.span>
        </AnimatePresence>
        <ChevronDown className={`size-3 transition-transform ${open ? "rotate-180" : ""}`} />
      </motion.button>
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.12 }}
            className="absolute z-50 mt-1 min-w-56 rounded-md border border-[var(--color-border)] bg-[var(--color-surface-raised)] shadow-[0_8px_32px_rgba(0,0,0,0.4)]"
          >
            {options.map((opt) => {
              const selected = opt.value === value;
              return (
                <button
                  key={opt.value}
                  type="button"
                  onClick={() => {
                    onChange(opt.value);
                    setOpen(false);
                  }}
                  className={cn(
                    "flex w-full cursor-pointer items-center justify-between gap-3 px-3 py-2 text-left text-sm transition-colors hover:bg-[var(--color-surface-hover)]",
                    selected ? "text-[var(--color-brand-blue)]" : opt.className ?? "text-[var(--color-text)]",
                  )}
                >
                  <span className="flex items-center gap-2 min-w-0">
                    {opt.icon}
                    <span className="truncate">{opt.label}</span>
                  </span>
                  <span className="flex items-center gap-2 shrink-0">
                    {opt.meta}
                    {selected && <Check className="size-3.5 text-[var(--color-brand-blue)]" />}
                  </span>
                </button>
              );
            })}
            {footer ? (
              <div className="border-t border-[var(--color-border)]">
                {typeof footer === "function" ? footer({ close: () => setOpen(false) }) : footer}
              </div>
            ) : null}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
