"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Check, Cpu, Plus, X } from "lucide-react";
import { MODELS } from "@/lib/constants";
import { loadCustomModels, addCustomModel } from "@/lib/custom-models";
import { isBedrockModelId } from "@/lib/model-utils";
import { ChipSelect, type ChipSelectOption } from "@/components/kit/chip-select";
import { BedrockBadge } from "@/components/shared/bedrock-badge";

export interface ModelChipProps {
  value: string;
  onChange: (model: string) => void;
}

// Hoisted to module scope — MODELS is a static import-time constant, so the
// derived value list never changes across renders or component instances.
const BUILTIN_MODELS = MODELS.map((m) => m.value);

/**
 * Pill-style model selector. Built-in options come from `MODELS` in
 * `@/lib/constants`; user-typed identifiers (e.g. Bedrock inference-profile
 * IDs) are appended via the "Add model" footer and persisted with
 * `custom-models.ts`.
 */
export function ModelChip({ value, onChange }: ModelChipProps) {
  const [custom, setCustom] = useState<string[]>([]);
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setCustom(loadCustomModels());
  }, []);

  useEffect(() => {
    if (adding) inputRef.current?.focus();
  }, [adding]);

  const allKnown = useMemo(
    () => [...new Set([...BUILTIN_MODELS, ...custom])],
    [custom],
  );

  // If the current value isn't in the known list (e.g. localStorage was
  // cleared but the active agent still references a custom model), surface
  // it at the top of the options so the chip displays the correct selection.
  const finalList = useMemo(
    () => (value && !allKnown.includes(value) ? [value, ...allKnown] : allKnown),
    [value, allKnown],
  );

  const options: ChipSelectOption[] = useMemo(
    () =>
      finalList.map((m) => ({
        value: m,
        label: (
          <span className="inline-flex items-center gap-1.5 min-w-0">
            <span className="truncate">{m}</span>
            {isBedrockModelId(m) && <BedrockBadge />}
          </span>
        ),
        icon: <Cpu className="size-3 shrink-0 text-[var(--color-text-muted)]" />,
      })),
    [finalList],
  );

  const trigger = (
    <>
      <Cpu className="size-3 text-[var(--color-brand-blue)]" />
      <span className="font-mono text-[var(--color-text)]">{shortModelLabel(value)}</span>
      {isBedrockModelId(value) && <BedrockBadge />}
    </>
  );

  const renderFooter = useCallback(({ close }: { close: () => void }) => (
    <div className="px-2 py-1.5">
      {adding ? (
        <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
          <input
            ref={inputRef}
            type="text"
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                const trimmed = draft.trim();
                if (trimmed) {
                  const next = addCustomModel(trimmed);
                  setCustom(next);
                  onChange(trimmed);
                }
                setAdding(false);
                setDraft("");
                close();
              }
              if (e.key === "Escape") {
                setAdding(false);
                setDraft("");
              }
            }}
            placeholder="model id (friendly or Bedrock)"
            className="flex-1 h-6 rounded px-2 text-[12px] font-mono bg-[var(--color-bg)] border border-[var(--color-border)] text-[var(--color-text)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-brand-blue)]/60"
          />
          <button
            type="button"
            onClick={() => {
              const trimmed = draft.trim();
              if (trimmed) {
                const next = addCustomModel(trimmed);
                setCustom(next);
                onChange(trimmed);
              }
              setAdding(false);
              setDraft("");
              close();
            }}
            className="flex items-center justify-center h-6 w-6 rounded text-green-400 hover:bg-[var(--color-surface-hover)] transition-colors"
          >
            <Check className="h-3.5 w-3.5" />
          </button>
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); setAdding(false); setDraft(""); }}
            className="flex items-center justify-center h-6 w-6 rounded text-[var(--color-text-muted)] hover:bg-[var(--color-surface-hover)] transition-colors"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      ) : (
        <button
          type="button"
          onClick={(e) => { e.stopPropagation(); setAdding(true); setDraft(""); }}
          className="flex w-full items-center gap-1.5 px-1 py-1 text-[12px] text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors rounded hover:bg-[var(--color-surface-hover)] cursor-pointer"
        >
          <Plus className="h-3 w-3" />
          Add model
        </button>
      )}
    </div>
  ), [adding, draft, onChange]);

  return <ChipSelect value={value} options={options} onChange={onChange} trigger={trigger} footer={renderFooter} />;
}

export function shortModelLabel(model: string): string {
  // claude-sonnet-4-6 → sonnet 4.6
  const m = model.match(/claude-(opus|sonnet|haiku)-(\d+)-(\d+)/);
  if (!m) return model;
  return `${m[1]} ${m[2]}.${m[3]}`;
}
