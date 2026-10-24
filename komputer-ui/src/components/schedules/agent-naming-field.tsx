"use client";

import { Label } from "@/components/kit/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/kit/select";
import { AGENT_NAMING_OPTIONS, exampleRunAgentName } from "@/lib/schedule-naming";
import type { ScheduleAgentNaming } from "@/lib/types";

type AgentNamingFieldProps = {
  value: ScheduleAgentNaming;
  onChange: (value: ScheduleAgentNaming) => void;
  scheduleName: string;
};

/** Picks whether a template schedule reuses one agent or creates one per run. */
export function AgentNamingField({ value, onChange, scheduleName }: AgentNamingFieldProps) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>Agent per run</Label>
      <Select value={value} onValueChange={(v) => v && onChange(v as ScheduleAgentNaming)}>
        <SelectTrigger className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {AGENT_NAMING_OPTIONS.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className="text-[11px] text-[var(--color-text-muted)]">
        Runs use <span className="font-mono">{exampleRunAgentName(scheduleName, value)}</span>
        {value === "Fixed" ? "." : " — a run is skipped while the previous run's agent is still working."}
      </p>
    </div>
  );
}
