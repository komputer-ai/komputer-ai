"use client";

import { useState, useEffect, useRef, useCallback, useMemo } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, Check, Plus, X } from "lucide-react";
import { Label } from "@/components/kit/label";
import { MODELS } from "@/lib/constants";
import { loadCustomModels, addCustomModel } from "@/lib/custom-models";
import { isBedrockModelId } from "@/lib/model-utils";
import { BedrockBadge } from "@/components/shared/bedrock-badge";

type ModelSelectorProps = {
  value: string;
  onChange: (model: string) => void;
  label?: string;
};

const BUILTIN_MODELS = MODELS.map((m) => m.value);

export function ModelSelector({ value, onChange, label = "Model" }: ModelSelectorProps) {
  const [custom, setCustom] = useState<string[]>([]);
  const [open, setOpen] = useState(false);
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState("");
  const dropdownRef = useRef<HTMLDivElement>(null);
  const addInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setCustom(loadCustomModels());
  }, []);

  useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setOpen(false);
        setAdding(false);
        setDraft("");
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") {
        setOpen(false);
        setAdding(false);
        setDraft("");
      }
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [open]);

  useEffect(() => {
    if (adding) addInputRef.current?.focus();
  }, [adding]);

  const handleSelect = useCallback((m: string) => {
    onChange(m);
    setOpen(false);
  }, [onChange]);

  const handleStartAdding = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    setAdding(true);
    setDraft("");
  }, []);

  const handleConfirmAdd = useCallback(() => {
    const trimmed = draft.trim();
    if (trimmed) {
      const next = addCustomModel(trimmed);
      setCustom(next);
      onChange(trimmed);
    }
    setAdding(false);
    setDraft("");
    setOpen(false);
  }, [draft, onChange]);

  const handleCancelAdd = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    setAdding(false);
    setDraft("");
  }, []);

  const handleAddKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === "Enter") { e.preventDefault(); handleConfirmAdd(); }
    if (e.key === "Escape") { e.preventDefault(); setAdding(false); setDraft(""); }
  }, [handleConfirmAdd]);

  // Built-in list + user-added, dedup. Value shows even if not in either list
  // (e.g. user pasted a value once then cleared their localStorage).
  const allKnown = useMemo(() => [...new Set([...BUILTIN_MODELS, ...custom])], [custom]);
  const showCurrentAsExtra = value && !allKnown.includes(value);

  return (
    <div className="flex flex-col gap-1.5">
      {label && <Label>{label}</Label>}
      <div className="relative" ref={dropdownRef}>
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="flex h-8 w-full items-center justify-between gap-2 rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 text-[13px] font-[family-name:var(--font-mono)] text-[var(--color-text)] shadow-[inset_0_2px_4px_rgba(0,0,0,0.2)] transition-all duration-150 hover:border-[var(--color-border-hover)] focus:outline-none cursor-pointer"
        >
          <span className="flex items-center gap-1.5 min-w-0">
            <span className="truncate">{value || "select a model"}</span>
            {isBedrockModelId(value) && <BedrockBadge />}
          </span>
          <ChevronDown className={`h-4 w-4 shrink-0 text-[var(--color-text-muted)] transition-transform duration-200 ${open ? "rotate-180" : ""}`} />
        </button>

        <AnimatePresence>
          {open && (
            <motion.div
              className="absolute z-50 w-full mt-1 py-1 rounded-[var(--radius-md)] bg-[var(--color-surface-raised)] border border-[var(--color-border)] shadow-[0_8px_32px_rgba(0,0,0,0.4),0_2px_8px_rgba(0,0,0,0.2)] overflow-y-auto max-h-60"
              initial={{ opacity: 0, y: -4, scale: 0.98 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -4, scale: 0.98 }}
              transition={{ duration: 0.12, ease: "easeOut" }}
            >
              {/* Add model row */}
              <div className="px-2 py-1 border-b border-[var(--color-border)] mb-1">
                {adding ? (
                  <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                    <input
                      ref={addInputRef}
                      type="text"
                      value={draft}
                      onChange={(e) => setDraft(e.target.value)}
                      onKeyDown={handleAddKeyDown}
                      placeholder="model id (friendly or Bedrock)"
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
                    className="flex w-full items-center gap-1.5 px-1 py-1 text-[12px] text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors rounded hover:bg-[var(--color-surface-hover)] cursor-pointer"
                  >
                    <Plus className="h-3 w-3" />
                    Add model
                  </button>
                )}
              </div>

              {/* Show the current value if it isn't in either list */}
              {showCurrentAsExtra && (
                <div
                  className="flex items-center justify-between gap-2 px-3 py-2 text-sm cursor-pointer transition-colors hover:bg-[var(--color-surface-hover)] text-[var(--color-brand-blue)]"
                  onMouseDown={(e) => { e.preventDefault(); handleSelect(value); }}
                >
                  <span className="flex items-center gap-1.5 min-w-0">
                    <span className="truncate font-[family-name:var(--font-mono)] text-[13px]">{value}</span>
                    {isBedrockModelId(value) && <BedrockBadge />}
                  </span>
                  <Check className="h-4 w-4 shrink-0 text-[var(--color-brand-blue)]" />
                </div>
              )}

              {/* Combined list */}
              {allKnown.map((m) => (
                <div
                  key={m}
                  className={`flex items-center justify-between gap-2 px-3 py-2 text-sm cursor-pointer transition-colors hover:bg-[var(--color-surface-hover)] ${value === m ? "text-[var(--color-brand-blue)]" : "text-[var(--color-text)]"}`}
                  onMouseDown={(e) => { e.preventDefault(); handleSelect(m); }}
                >
                  <span className="flex items-center gap-1.5 min-w-0">
                    <span className="truncate font-[family-name:var(--font-mono)] text-[13px]">{m}</span>
                    {isBedrockModelId(m) && <BedrockBadge />}
                  </span>
                  {value === m && <Check className="h-4 w-4 shrink-0 text-[var(--color-brand-blue)]" />}
                </div>
              ))}
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
