"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, Check, Plus, X } from "lucide-react";
import { Label } from "@/components/kit/label";
import { listNamespaces } from "@/lib/api";

type NamespaceSelectorProps = {
  value: string;
  onChange: (ns: string) => void;
  label?: string;
};

export function NamespaceSelector({ value, onChange, label = "Namespace" }: NamespaceSelectorProps) {
  const [namespaces, setNamespaces] = useState<string[]>(["default"]);
  const [open, setOpen] = useState(false);
  const [adding, setAdding] = useState(false);
  const [draftNs, setDraftNs] = useState("");
  const dropdownRef = useRef<HTMLDivElement>(null);
  const addInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    listNamespaces()
      .then((res) => {
        const sorted = [...new Set(["default", ...(res.namespaces || [])])].sort();
        setNamespaces(sorted);
      })
      .catch(() => {});
  }, []);

  // Close on click outside
  useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setOpen(false);
        setAdding(false);
        setDraftNs("");
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  // Focus input when add mode opens
  useEffect(() => {
    if (adding) addInputRef.current?.focus();
  }, [adding]);

  const handleSelect = useCallback((ns: string) => {
    onChange(ns);
    setOpen(false);
  }, [onChange]);

  const handleStartAdding = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    setAdding(true);
    setDraftNs("");
  }, []);

  const handleConfirmAdd = useCallback(() => {
    const trimmed = draftNs.trim();
    if (trimmed) {
      if (!namespaces.includes(trimmed)) {
        setNamespaces((prev) => [...prev, trimmed].sort());
      }
      onChange(trimmed);
    }
    setAdding(false);
    setDraftNs("");
    setOpen(false);
  }, [draftNs, namespaces, onChange]);

  const handleCancelAdd = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    setAdding(false);
    setDraftNs("");
  }, []);

  const handleAddKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === "Enter") { e.preventDefault(); handleConfirmAdd(); }
    if (e.key === "Escape") { e.preventDefault(); setAdding(false); setDraftNs(""); }
  }, [handleConfirmAdd]);

  return (
    <div className="flex flex-col gap-1.5">
      {label && <Label>{label}</Label>}
      <div className="relative" ref={dropdownRef}>
        {/* Trigger */}
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="flex h-8 w-full items-center justify-between rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 text-[13px] font-[family-name:var(--font-mono)] text-[var(--color-text)] shadow-[inset_0_2px_4px_rgba(0,0,0,0.2)] transition-all duration-150 hover:border-[var(--color-border-hover)] focus:outline-none cursor-pointer"
        >
          <span className="truncate">{value || "default"}</span>
          <ChevronDown className={`h-4 w-4 shrink-0 text-[var(--color-text-muted)] transition-transform duration-200 ${open ? "rotate-180" : ""}`} />
        </button>

        {/* Dropdown */}
        <AnimatePresence>
          {open && (
            <motion.div
              className="absolute z-50 w-full mt-1 py-1 rounded-[var(--radius-md)] bg-[var(--color-surface-raised)] border border-[var(--color-border)] shadow-[0_8px_32px_rgba(0,0,0,0.4),0_2px_8px_rgba(0,0,0,0.2)] overflow-y-auto max-h-60"
              initial={{ opacity: 0, y: -4, scale: 0.98 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -4, scale: 0.98 }}
              transition={{ duration: 0.12, ease: "easeOut" }}
            >
              {/* Add namespace row */}
              <div className="px-2 py-1 border-b border-[var(--color-border)] mb-1">
                {adding ? (
                  <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                    <input
                      ref={addInputRef}
                      type="text"
                      value={draftNs}
                      onChange={(e) => setDraftNs(e.target.value)}
                      onKeyDown={handleAddKeyDown}
                      placeholder="namespace name"
                      className="flex-1 h-6 rounded px-2 text-[12px] font-[family-name:var(--font-mono)] bg-[var(--color-bg)] border border-[var(--color-border)] text-[var(--color-text)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-brand-blue)]/60"
                    />
                    <button
                      type="button"
                      onClick={handleConfirmAdd}
                      className="flex items-center justify-center h-6 w-6 rounded text-green-400 hover:bg-[var(--color-surface-hover)] transition-colors"
                    >
                      <Check className="h-3.5 w-3.5" />
                    </button>
                    <button
                      type="button"
                      onClick={handleCancelAdd}
                      className="flex items-center justify-center h-6 w-6 rounded text-[var(--color-text-muted)] hover:bg-[var(--color-surface-hover)] transition-colors"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ) : (
                  <button
                    type="button"
                    onClick={handleStartAdding}
                    className="flex w-full items-center gap-1.5 px-1 py-1 text-[12px] text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors rounded hover:bg-[var(--color-surface-hover)]"
                  >
                    <Plus className="h-3 w-3" />
                    Add namespace
                  </button>
                )}
              </div>

              {/* Namespace list */}
              {namespaces.map((ns) => (
                <div
                  key={ns}
                  className={`flex items-center justify-between px-3 py-2 text-sm cursor-pointer transition-colors hover:bg-[var(--color-surface-hover)] ${value === ns ? "text-[var(--color-brand-blue)]" : "text-[var(--color-text)]"}`}
                  onMouseDown={(e) => { e.preventDefault(); handleSelect(ns); }}
                >
                  <span className="truncate font-[family-name:var(--font-mono)] text-[13px]">{ns}</span>
                  {value === ns && <Check className="h-4 w-4 shrink-0 text-[var(--color-brand-blue)]" />}
                </div>
              ))}
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
