"use client";

import { Button } from "@/components/kit/button";
import { AgentFieldsForm, type AgentFormValues } from "./agent-fields-form";

export interface SoloModeFormProps {
  values: AgentFormValues;
  onChange: (v: AgentFormValues) => void;
  active: boolean;
  submitting: boolean;
  error: string | null;
  onSubmit: () => void;
  onCancel: () => void;
  submitLabel?: string;
}

export function SoloModeForm({
  values,
  onChange,
  active,
  submitting,
  error,
  onSubmit,
  onCancel,
  submitLabel = "Create Agent",
}: SoloModeFormProps) {
  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    onSubmit();
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col flex-1 min-h-0">
      <div className="flex-1 min-h-0 overflow-y-auto pr-1">
        <AgentFieldsForm
          values={values}
          onChange={onChange}
          active={active}
          idPrefix="solo"
        />
        {error && <p className="text-sm text-red-400 mt-3">{error}</p>}
      </div>

      <div className="mt-4 shrink-0 flex justify-end gap-2">
        <Button variant="secondary" type="button" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={submitting}>
          {submitting ? "Creating..." : submitLabel}
        </Button>
      </div>
    </form>
  );
}
