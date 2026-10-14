"use client";

import { useRef } from "react";
import { ChevronRight, Lock, Plus, Trash2 } from "lucide-react";
import { Input } from "@/components/kit/input";
import { cn } from "@/lib/utils";
import type { LabelRow, LabelRowErrors } from "@/lib/label-edit";

interface LabelsEditorProps {
  rows: LabelRow[];
  errors: LabelRowErrors[];
  system: [string, string][];
  /** Original user labels, used to mark new/changed rows. */
  original: Record<string, string>;
  onChange: (rows: LabelRow[]) => void;
}

// Key/value table for an agent's user labels. Edits are staged; the parent
// form sends them with Save Changes. komputer.ai/* labels are read-only.
export function LabelsEditor({ rows, errors, system, original, onChange }: LabelsEditorProps) {
  const tableRef = useRef<HTMLDivElement>(null);

  const update = (i: number, field: keyof LabelRow, value: string) =>
    onChange(rows.map((r, j) => (j === i ? { ...r, [field]: value } : r)));

  const addRow = () => {
    onChange([...rows, { key: "", value: "" }]);
    // Focus the new row's key input once it renders.
    requestAnimationFrame(() => {
      const inputs = tableRef.current?.querySelectorAll<HTMLInputElement>("input[data-field=key]");
      inputs?.[inputs.length - 1]?.focus();
    });
  };

  return (
    <div className="flex flex-col gap-2">
      {rows.length > 0 ? (
        <div ref={tableRef} className="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)_auto] items-start gap-x-1.5 gap-y-1.5">
          <span className="text-[11px] text-[var(--color-text-muted)]">Key</span>
          <span />
          <span className="text-[11px] text-[var(--color-text-muted)]">Value</span>
          <span />
          {rows.map((r, i) => {
            const err = errors[i] ?? {};
            const changed = r.key.trim() !== "" && original[r.key.trim()] !== r.value.trim();
            return (
              <div key={i} className="contents">
                <div className="flex flex-col gap-0.5">
                  <Input
                    data-field="key"
                    aria-label={`Label ${i + 1} key`}
                    placeholder="key"
                    value={r.key}
                    onChange={(e) => update(i, "key", e.target.value)}
                    className={cn("h-7 text-[12px]", err.key && "border-[var(--color-status-error)]", !err.key && changed && "border-[var(--color-status-success)]/60")}
                  />
                  {err.key && <span className="text-[11px] text-[var(--color-status-error)]">{err.key}</span>}
                </div>
                <span className="font-mono text-[12px] leading-7 text-[var(--color-text-muted)]">=</span>
                <div className="flex flex-col gap-0.5">
                  <Input
                    aria-label={`Label ${i + 1} value`}
                    placeholder="value"
                    value={r.value}
                    onChange={(e) => update(i, "value", e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && i === rows.length - 1) {
                        e.preventDefault();
                        addRow();
                      }
                    }}
                    className={cn("h-7 text-[12px]", err.value && "border-[var(--color-status-error)]", !err.value && changed && "border-[var(--color-status-success)]/60")}
                  />
                  {err.value && <span className="text-[11px] text-[var(--color-status-error)]">{err.value}</span>}
                </div>
                <button
                  type="button"
                  aria-label={`Remove label ${r.key || i + 1}`}
                  onClick={() => onChange(rows.filter((_, j) => j !== i))}
                  className="inline-flex size-7 items-center justify-center rounded-[var(--radius-sm)] text-[var(--color-text-muted)] transition-colors hover:bg-[var(--color-status-error)]/10 hover:text-[var(--color-status-error)] cursor-pointer"
                >
                  <Trash2 className="size-3.5" />
                </button>
              </div>
            );
          })}
        </div>
      ) : (
        <p className="text-[12px] text-[var(--color-text-muted)]">No labels yet.</p>
      )}

      <div>
        <button
          type="button"
          onClick={addRow}
          className="inline-flex items-center gap-1 rounded-[var(--radius-sm)] border border-[var(--color-border)] px-2 py-1 text-[12px] text-[var(--color-text-secondary)] transition-colors hover:border-[var(--color-border-hover)] hover:text-[var(--color-text)] cursor-pointer"
        >
          <Plus className="size-3" /> Add label
        </button>
      </div>

      {system.length > 0 && (
        <details className="group">
          <summary className="flex cursor-pointer list-none items-center gap-1 text-[12px] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] [&::-webkit-details-marker]:hidden">
            <ChevronRight className="size-3 transition-transform group-open:rotate-90" />
            {system.length} system label{system.length === 1 ? "" : "s"} (read-only)
          </summary>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {system.map(([k, v]) => (
              <span
                key={k}
                className="inline-flex items-center gap-1 rounded-[var(--radius-sm)] border border-[var(--color-brand-blue)]/40 bg-[var(--color-brand-blue)]/10 px-2 py-0.5 font-mono text-[11px] text-[var(--color-brand-blue-light)]"
              >
                <Lock className="size-2.5" />
                {k}={v}
              </span>
            ))}
          </div>
        </details>
      )}
    </div>
  );
}
