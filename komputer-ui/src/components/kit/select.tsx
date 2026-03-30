"use client";

import {
  createContext,
  useContext,
  useState,
  useRef,
  useEffect,
  useCallback,
  type ReactNode,
  type RefObject,
  type ButtonHTMLAttributes,
  type HTMLAttributes,
} from "react";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, Check } from "lucide-react";
import { cn } from "@/lib/utils";

/* ------------------------------------------------------------------ */
/*  Context                                                           */
/* ------------------------------------------------------------------ */

interface SelectContextType {
  open: boolean;
  setOpen: (open: boolean) => void;
  value: string;
  onValueChange: (value: string) => void;
  triggerRef: RefObject<HTMLButtonElement | null>;
  labels: Map<string, string>;
  registerLabel: (value: string, label: string) => void;
  highlightedIndex: number;
  setHighlightedIndex: (i: number) => void;
  itemValues: RefObject<string[]>;
}

const SelectContext = createContext<SelectContextType | null>(null);

function useSelectContext() {
  const ctx = useContext(SelectContext);
  if (!ctx) throw new Error("Select components must be used within <Select>");
  return ctx;
}

/* ------------------------------------------------------------------ */
/*  Select (root)                                                     */
/* ------------------------------------------------------------------ */

interface SelectProps {
  value: string;
  onValueChange: (value: string) => void;
  children: ReactNode;
}

export function Select({ value, onValueChange, children }: SelectProps) {
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [labels, setLabels] = useState(() => new Map<string, string>());
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const itemValues = useRef<string[]>([]);

  const registerLabel = useCallback(
    (val: string, label: string) => {
      setLabels((prev) => {
        if (prev.get(val) === label) return prev;
        const next = new Map(prev);
        next.set(val, label);
        return next;
      });
    },
    []
  );

  return (
    <SelectContext.Provider
      value={{
        open,
        setOpen,
        value,
        onValueChange,
        triggerRef,
        labels,
        registerLabel,
        highlightedIndex,
        setHighlightedIndex,
        itemValues,
      }}
    >
      <div className="relative">{children}</div>
    </SelectContext.Provider>
  );
}

/* ------------------------------------------------------------------ */
/*  SelectTrigger                                                     */
/* ------------------------------------------------------------------ */

interface SelectTriggerProps extends ButtonHTMLAttributes<HTMLButtonElement> {}

export function SelectTrigger({
  className,
  children,
  ...props
}: SelectTriggerProps) {
  const { open, setOpen, triggerRef, highlightedIndex, setHighlightedIndex, itemValues, onValueChange } =
    useSelectContext();

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const items = itemValues.current;
      if (!items.length) return;

      if (e.key === "ArrowDown") {
        e.preventDefault();
        if (!open) {
          setOpen(true);
          setHighlightedIndex(0);
        } else {
          setHighlightedIndex(
            highlightedIndex < items.length - 1 ? highlightedIndex + 1 : 0
          );
        }
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        if (open) {
          setHighlightedIndex(
            highlightedIndex > 0 ? highlightedIndex - 1 : items.length - 1
          );
        }
      } else if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        if (open && highlightedIndex >= 0 && highlightedIndex < items.length) {
          onValueChange(items[highlightedIndex]);
          setOpen(false);
        } else if (!open) {
          setOpen(true);
          setHighlightedIndex(0);
        }
      } else if (e.key === "Escape") {
        e.preventDefault();
        setOpen(false);
      }
    },
    [open, setOpen, highlightedIndex, setHighlightedIndex, itemValues, onValueChange]
  );

  return (
    <button
      ref={triggerRef}
      type="button"
      role="combobox"
      aria-expanded={open}
      aria-haspopup="listbox"
      className={cn(
        "flex items-center justify-between w-full h-8 px-3 rounded-[var(--radius-sm)] text-[13px] font-[family-name:var(--font-mono)]",
        "bg-[var(--color-bg)] border border-[var(--color-border)] text-[var(--color-text)]",
        "shadow-[inset_0_2px_4px_rgba(0,0,0,0.2)]",
        "transition-all duration-150 cursor-pointer",
        "hover:border-[var(--color-border-hover)]",
        "focus:outline-none focus:border-[var(--color-brand-blue)]/60 focus:shadow-[inset_0_2px_4px_rgba(0,0,0,0.2),0_0_0_2px_var(--color-brand-blue-glow)]",
        open &&
          "border-[var(--color-brand-blue)]/60 shadow-[inset_0_2px_4px_rgba(0,0,0,0.2),0_0_0_2px_var(--color-brand-blue-glow)]",
        className
      )}
      onClick={() => {
        setOpen(!open);
        if (!open) setHighlightedIndex(-1);
      }}
      onKeyDown={handleKeyDown}
      {...props}
    >
      {children}
      <ChevronDown
        className={cn(
          "h-4 w-4 text-[var(--color-text-muted)] transition-transform duration-200",
          open && "rotate-180"
        )}
      />
    </button>
  );
}

/* ------------------------------------------------------------------ */
/*  SelectValue                                                       */
/* ------------------------------------------------------------------ */

export function SelectValue({ placeholder }: { placeholder?: string }) {
  const { value, labels } = useSelectContext();
  const display = labels.get(value) || value;

  return (
    <span
      className={cn("truncate", !value && "text-[var(--color-text-muted)]")}
    >
      {value ? display : placeholder || "Select..."}
    </span>
  );
}

/* ------------------------------------------------------------------ */
/*  SelectContent                                                     */
/* ------------------------------------------------------------------ */

interface SelectContentProps extends HTMLAttributes<HTMLDivElement> {}

export function SelectContent({ className, children }: SelectContentProps) {
  const { open, setOpen, triggerRef } = useSelectContext();
  const contentRef = useRef<HTMLDivElement>(null);

  // Close on click outside
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
  }, [open, setOpen, triggerRef]);

  // Close on Escape
  useEffect(() => {
    if (!open) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [open, setOpen]);

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          ref={contentRef}
          role="listbox"
          className={cn(
            "absolute z-50 w-full mt-1 py-1 rounded-[var(--radius-md)]",
            "bg-[var(--color-surface-raised)] border border-[var(--color-border)]",
            "shadow-[0_8px_32px_rgba(0,0,0,0.4),0_2px_8px_rgba(0,0,0,0.2)]",
            "overflow-hidden",
            className
          )}
          initial={{ opacity: 0, y: -4, scale: 0.98 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: -4, scale: 0.98 }}
          transition={{ duration: 0.12, ease: "easeOut" }}
        >
          {children}
        </motion.div>
      )}
    </AnimatePresence>
  );
}

/* ------------------------------------------------------------------ */
/*  SelectItem                                                        */
/* ------------------------------------------------------------------ */

interface SelectItemProps extends HTMLAttributes<HTMLDivElement> {
  value: string;
}

export function SelectItem({
  value: itemValue,
  className,
  children,
  ...props
}: SelectItemProps) {
  const {
    value,
    onValueChange,
    setOpen,
    registerLabel,
    highlightedIndex,
    setHighlightedIndex,
    itemValues,
  } = useSelectContext();

  const isSelected = value === itemValue;

  // Register label on mount and when children change
  useEffect(() => {
    const label =
      typeof children === "string"
        ? children
        : typeof children === "number"
          ? String(children)
          : itemValue;
    registerLabel(itemValue, label);
  }, [itemValue, children, registerLabel]);

  // Register item value for keyboard nav
  useEffect(() => {
    const items = itemValues.current;
    if (!items.includes(itemValue)) {
      items.push(itemValue);
    }
    return () => {
      const idx = items.indexOf(itemValue);
      if (idx !== -1) items.splice(idx, 1);
    };
  }, [itemValue, itemValues]);

  const index = itemValues.current.indexOf(itemValue);
  const isHighlighted = index === highlightedIndex;

  return (
    <div
      role="option"
      aria-selected={isSelected}
      className={cn(
        "flex items-center justify-between px-3 py-2 text-sm cursor-pointer transition-colors",
        "hover:bg-[var(--color-surface-hover)]",
        isHighlighted && "bg-[var(--color-surface-hover)]",
        isSelected
          ? "text-[var(--color-brand-blue)]"
          : "text-[var(--color-text)]",
        className
      )}
      onClick={() => {
        onValueChange(itemValue);
        setOpen(false);
      }}
      onMouseEnter={() => setHighlightedIndex(index)}
      {...props}
    >
      <span className="truncate">{children}</span>
      {isSelected && (
        <Check className="h-4 w-4 shrink-0 text-[var(--color-brand-blue)]" />
      )}
    </div>
  );
}
