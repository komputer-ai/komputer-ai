"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { useRouter } from "next/navigation";
import { Building2, Clock } from "lucide-react";
import { StatusBadge } from "@/components/shared/status-badge";

/* ------------------------------------------------------------------ */
/*  Status-dot color (mirrors StatusBadge logic)                      */
/* ------------------------------------------------------------------ */

const dotColorMap: Record<string, string> = {
  Running: "bg-[var(--color-brand-blue)]",
  Active: "bg-[var(--color-brand-blue)]",
  InProgress: "bg-[var(--color-brand-blue)]",
  Sleeping: "bg-amber-400",
  Suspended: "bg-amber-400",
  Failed: "bg-red-400",
  Error: "bg-red-400",
  Pending: "bg-blue-400",
  Succeeded: "bg-green-400",
  Complete: "bg-green-400",
};

function statusDotClass(status: string) {
  return dotColorMap[status] ?? "bg-[var(--color-text-secondary)]";
}

/* ------------------------------------------------------------------ */
/*  AgentNode                                                          */
/* ------------------------------------------------------------------ */

export type AgentNodeData = {
  label: string;
  status: string;
  model: string;
};

function AgentNodeComponent({ data }: NodeProps) {
  const router = useRouter();
  const { label, status, model } = data as unknown as AgentNodeData;

  return (
    <div
      className="cursor-pointer rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-3 shadow-md transition-shadow hover:shadow-lg"
      style={{ minWidth: 160 }}
      onClick={() => router.push(`/agents/${label}`)}
    >
      <Handle type="target" position={Position.Top} className="!bg-[var(--color-brand-blue)]" />

      <div className="flex items-center gap-2">
        <span
          className={`inline-block h-2 w-2 shrink-0 rounded-full ${statusDotClass(status)}`}
        />
        <span className="text-sm font-medium text-[var(--color-text)] truncate">
          {label}
        </span>
      </div>

      <p className="mt-1 text-[10px] text-[var(--color-text-secondary)]">
        {model || "claude"}
      </p>

      <Handle type="source" position={Position.Bottom} className="!bg-[var(--color-brand-blue)]" />
    </div>
  );
}

export const AgentNode = memo(AgentNodeComponent);

/* ------------------------------------------------------------------ */
/*  OfficeNode                                                         */
/* ------------------------------------------------------------------ */

export type OfficeNodeData = {
  label: string;
  phase: string;
  agentCount: number;
};

function OfficeNodeComponent({ data }: NodeProps) {
  const router = useRouter();
  const { label, phase, agentCount } = data as unknown as OfficeNodeData;

  return (
    <div
      className="cursor-pointer rounded-lg border border-[var(--color-border)] bg-[#1e2d3d] px-5 py-3.5 shadow-md transition-shadow hover:shadow-lg"
      style={{ minWidth: 180 }}
      onClick={() => router.push(`/offices/${label}`)}
    >
      <Handle type="target" position={Position.Top} className="!bg-[var(--color-brand-blue)]" />

      <div className="flex items-center gap-2">
        <Building2 className="h-4 w-4 shrink-0 text-[var(--color-brand-blue)]" />
        <span className="text-sm font-medium text-[var(--color-text)] truncate">
          {label}
        </span>
      </div>

      <div className="mt-2 flex items-center justify-between gap-3">
        <StatusBadge status={phase} size="sm" />
        <span className="text-[10px] text-[var(--color-text-secondary)]">
          {agentCount} agent{agentCount !== 1 ? "s" : ""}
        </span>
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-[var(--color-brand-blue)]" />
    </div>
  );
}

export const OfficeNode = memo(OfficeNodeComponent);

/* ------------------------------------------------------------------ */
/*  ScheduleNode                                                       */
/* ------------------------------------------------------------------ */

export type ScheduleNodeData = {
  label: string;
  cron: string;
  phase: string;
};

function ScheduleNodeComponent({ data }: NodeProps) {
  const router = useRouter();
  const { label, cron, phase } = data as unknown as ScheduleNodeData;

  return (
    <div
      className="cursor-pointer rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-3 shadow-md transition-shadow hover:shadow-lg"
      style={{ minWidth: 170 }}
      onClick={() => router.push(`/schedules/${label}`)}
    >
      <Handle type="target" position={Position.Top} className="!bg-cyan-400" />

      <div className="flex items-center gap-2">
        <Clock className="h-4 w-4 shrink-0 text-cyan-400" />
        <span className="text-sm font-medium text-[var(--color-text)] truncate">
          {label}
        </span>
      </div>

      <p className="mt-1 text-[10px] text-[var(--color-text-secondary)]">{cron}</p>

      <div className="mt-1.5">
        <StatusBadge status={phase} size="sm" />
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-cyan-400" />
    </div>
  );
}

export const ScheduleNode = memo(ScheduleNodeComponent);
