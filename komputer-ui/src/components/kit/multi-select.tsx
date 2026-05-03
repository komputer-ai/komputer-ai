"use client";

import {
  useState,
  useRef,
  useEffect,
  useMemo,
  useCallback,
  useId,
  type ReactNode,
} from "react";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, Check, Search } from "lucide-react";
import { cn } from "@/lib/utils";

export interface MultiSelectOption {
  /** Unique value stored in the selection array */
  value: string;
  /** Primary label shown in the trigger and list */
  label: string;
  /** Optional secondary text shown as a subtle pill (e.g. namespace) */
  secondary?: string | null;
  /** Optional terms added to the search index (e.g. namespace, full ref) */
  searchTerms?: string[];
  /** Optional leading icon rendered in the dropdown row (e.g. provider logo) */
  icon?: React.ReactNode;
}

export interface MultiSelectProps {
  options: MultiSelectOption[];
  value: string[];
  onChange: (next: string[]) => void;
  placeholder?: string;
  emptyText?: string;
  searchPlaceholder?: string;
  /** Element rendered above the list inside the panel (e.g. "Show all" toggle) */
  headerExtra?: ReactNode;
  /** Element rendered below the list inside the panel (e.g. "New Secret" button) */
  footerExtra?: ReactNode;
  /** Trigger label noun used in the count display, e.g. "secrets" -> "3 secrets selected" */
  noun?: string;
  className?: string;
  disabled?: boolean;
}

export function MultiSelect({
  options,
  value,
  onChange,
  placeholder = "Select...",
  emptyText = "No items available",
  searchPlaceholder = "Search...",
  headerExtra,
  footerExtra,
  noun,
  className,
  disabled,
}: MultiSelectProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  // Raw highlighted index — clamped via highlightedIndex below so it stays valid
  // when the filtered list shrinks without needing a sync effect.
  const [rawHighlightedIndex, setRawHighlightedIndex] = useState(-1);
  const [openUpward, setOpenUpward] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const listboxId = useId();

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return options;
    return options.filter((o) => {
      if (o.label.toLowerCase().includes(q)) return true;
      if (o.secondary?.toLowerCase().includes(q)) return true;
      if (o.searchTerms?.some((t) => t.toLowerCase().includes(q))) return true;
      return false;
    });
  }, [options, query]);

  const highlightedIndex =
    rawHighlightedIndex >= 0 && rawHighlightedIndex < filtered.length
      ? rawHighlightedIndex
      : filtered.length > 0
        ? 0
        : -1;
  const setHighlightedIndex = setRawHighlightedIndex;

  // Reset search and focus when opening; reset on close.
  // Resetting query/highlight here is intentional: the open state IS the
  // external system (popover lifecycle). Use functional updates to avoid
  // the eslint set-state-in-effect rule.
  useEffect(() => {
    if (!open) {
      setQuery((q) => (q === "" ? q : ""));
      setRawHighlightedIndex((i) => (i === -1 ? i : -1));
      return;
    }
    // Position relative to viewport
    if (triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect();
      const spaceBelow = window.innerHeight - rect.bottom;
      setOpenUpward((cur) => {
        const next = spaceBelow < 360;
        return cur === next ? cur : next;
      });
    }
    const t = setTimeout(() => searchRef.current?.focus(), 10);
    return () => clearTimeout(t);
  }, [open]);

  // Outside click
  useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      if (
        contentRef.current &&
        !contentRef.current.contains(e.target as Node) &&
        triggerRef.current &&
        !triggerRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  // Escape closes
  useEffect(() => {
    if (!open) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [open]);

  const toggle = useCallback(
    (v: string) => {
      onChange(value.includes(v) ? value.filter((x) => x !== v) : [...value, v]);
    },
    [value, onChange]
  );

  const handleSearchKey = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setHighlightedIndex((i) => (filtered.length === 0 ? -1 : (i + 1) % filtered.length));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setHighlightedIndex((i) =>
        filtered.length === 0 ? -1 : (i - 1 + filtered.length) % filtered.length
      );
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (highlightedIndex >= 0 && highlightedIndex < filtered.length) {
        toggle(filtered[highlightedIndex].value);
      }
    }
  };

  const triggerLabel = (() => {
    if (value.length === 0) return placeholder;
    if (value.length === 1) {
      const opt = options.find((o) => o.value === value[0]);
      return opt?.label ?? value[0];
    }
    if (noun) return `${value.length} ${noun} selected`;
    return `${value.length} selected`;
  })();

  return (
    <div className={cn("relative", className)}>
      <button
        ref={triggerRef}
        type="button"
        role="combobox"
        aria-expanded={open}
        aria-haspopup="listbox"
        aria-controls={listboxId}
        disabled={disabled}
        className={cn(
          "flex items-center justify-between w-full h-8 px-3 rounded-[var(--radius-sm)] text-[13px] font-[family-name:var(--font-mono)]",
          "bg-[var(--color-bg)] border border-[var(--color-border)] text-[var(--color-text)]",
          "shadow-[inset_0_2px_4px_rgba(0,0,0,0.2)]",
          "transition-all duration-150 cursor-pointer",
          "hover:border-[var(--color-border-hover)]",
          "focus:outline-none focus:border-[var(--color-brand-blue)]/60 focus:shadow-[inset_0_2px_4px_rgba(0,0,0,0.2),0_0_0_2px_var(--color-brand-blue-glow)]",
          "disabled:opacity-40 disabled:cursor-not-allowed",
          open &&
            "border-[var(--color-brand-blue)]/60 shadow-[inset_0_2px_4px_rgba(0,0,0,0.2),0_0_0_2px_var(--color-brand-blue-glow)]"
        )}
        onClick={() => !disabled && setOpen((o) => !o)}
      >
        <span className={cn("truncate", value.length === 0 && "text-[var(--color-text-muted)]")}>
          {triggerLabel}
        </span>
        <ChevronDown
          className={cn(
            "h-4 w-4 shrink-0 text-[var(--color-text-muted)] transition-transform duration-200",
            open && "rotate-180"
          )}
        />
      </button>

      <AnimatePresence>
        {open && (
          <motion.div
            ref={contentRef}
            id={listboxId}
            role="listbox"
            aria-multiselectable="true"
            className={cn(
              "absolute z-50 w-full rounded-[var(--radius-md)]",
              "bg-[var(--color-surface-raised)] border border-[var(--color-border)]",
              "shadow-[0_8px_32px_rgba(0,0,0,0.4),0_2px_8px_rgba(0,0,0,0.2)]",
              "flex flex-col overflow-hidden",
              openUpward ? "bottom-full mb-1" : "mt-1"
            )}
            initial={{ opacity: 0, y: openUpward ? 4 : -4, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: openUpward ? 4 : -4, scale: 0.98 }}
            transition={{ duration: 0.12, ease: "easeOut" }}
          >
            {/* Search */}
            <div className="relative border-b border-[var(--color-border)]">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-[var(--color-text-muted)] pointer-events-none" />
              <input
                ref={searchRef}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={handleSearchKey}
                placeholder={searchPlaceholder}
                className={cn(
                  "w-full h-8 pl-8 pr-2 text-[13px] font-[family-name:var(--font-mono)]",
                  "bg-transparent text-[var(--color-text)] placeholder:text-[var(--color-text-muted)]",
                  "focus:outline-none"
                )}
              />
            </div>

            {headerExtra && (
              <div className="px-2 py-1.5 border-b border-[var(--color-border)]">
                {headerExtra}
              </div>
            )}

            {/* List */}
            <div className="overflow-y-auto max-h-60 py-1">
              {filtered.length === 0 ? (
                <div className="px-3 py-2 text-xs text-[var(--color-text-muted)]">
                  {query ? "No matches" : emptyText}
                </div>
              ) : (
                filtered.map((opt, idx) => {
                  const selected = value.includes(opt.value);
                  const highlighted = idx === highlightedIndex;
                  return (
                    <div
                      key={opt.value}
                      role="option"
                      aria-selected={selected}
                      className={cn(
                        "flex items-center justify-between gap-2 px-3 py-2 text-sm cursor-pointer transition-colors",
                        "hover:bg-[var(--color-surface-hover)]",
                        highlighted && "bg-[var(--color-surface-hover)]",
                        selected
                          ? "text-[var(--color-brand-blue)]"
                          : "text-[var(--color-text)]"
                      )}
                      onClick={() => toggle(opt.value)}
                      onMouseEnter={() => setHighlightedIndex(idx)}
                    >
                      <span className="flex items-center gap-2 min-w-0 flex-1">
                        {opt.icon && (
                          <span className="flex h-4 w-4 shrink-0 items-center justify-center">
                            {opt.icon}
                          </span>
                        )}
                        <span className="truncate">{opt.label}</span>
                        {opt.secondary && (
                          <span className="text-[10px] text-[var(--color-brand-blue-light)] shrink-0">
                            {opt.secondary}
                          </span>
                        )}
                      </span>
                      {selected && (
                        <Check className="h-4 w-4 shrink-0 text-[var(--color-brand-blue)]" />
                      )}
                    </div>
                  );
                })
              )}
            </div>

            {footerExtra && (
              <div className="px-2 py-1.5 border-t border-[var(--color-border)]">
                {footerExtra}
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
