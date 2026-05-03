"use client";

import { memo, useState } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { useRouter } from "next/navigation";
import { Building2, Clock } from "lucide-react";
import { StatusBadge } from "@/components/shared/status-badge";

function TooltipRow({ label, value, color }: { label: string; value?: string | number; color?: string }) {
  if (!value && value !== 0) return null;
  return (
    <div className="flex justify-between gap-6">
      <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">{label}</span>
      <span className="text-[11px] font-medium text-right" style={color ? { color } : undefined}>
        {value}
      </span>
    </div>
  );
}

function TooltipHeader({ icon, title, subtitle }: { icon: React.ReactNode; title: string; subtitle?: string }) {
  return (
    <div className="flex items-center gap-2 pb-2 mb-2 border-b border-[var(--color-border)]">
      {icon}
      <div className="min-w-0">
        <p className="text-[12px] font-semibold text-[var(--color-text)] truncate">{title}</p>
        {subtitle && <p className="text-[10px] text-[var(--color-text-muted)]">{subtitle}</p>}
      </div>
    </div>
  );
}

const statusColorMap: Record<string, string> = {
  Running: "#34D399", Active: "#34D399", InProgress: "#3f85d9",
  Sleeping: "#FBBF24", Suspended: "#FBBF24", Pending: "#60A5FA",
  Failed: "#F87171", Error: "#F87171",
  Succeeded: "#34D399", Complete: "#34D399",
  Deleted: "#F87171",
};

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
  Deleted: "bg-red-400/50",
};

function statusDotClass(status: string) {
  return dotColorMap[status] ?? "bg-[var(--color-text-secondary)]";
}

const handleVisible = "!bg-[var(--color-brand-blue)]";
const handleHidden = "!opacity-0 !w-0 !h-0";

/* ------------------------------------------------------------------ */
/*  AgentNode                                                          */
/* ------------------------------------------------------------------ */

export type AgentNodeData = {
  label: string;
  status: string;
  model: string;
  namespace?: string;
  taskStatus?: string;
  lifecycle?: string;
  totalCostUSD?: string;
  hasIncoming?: boolean;
  hasOutgoing?: boolean;
  nodeWidth?: number;
};

function AgentNodeComponent({ data }: NodeProps) {
  const router = useRouter();
  const { label, status, model, namespace, taskStatus, lifecycle, totalCostUSD, hasIncoming, hasOutgoing, nodeWidth } = data as unknown as AgentNodeData;
  const isDeleted = status === "Deleted";
  const [hovered, setHovered] = useState(false);

  return (
    <div
      className={`relative cursor-pointer rounded-lg border px-4 py-3 transition-shadow hover:shadow-[0_4px_16px_rgba(var(--shadow-color),var(--shadow-strength)),inset_0_1px_0_var(--color-border-light)] ${
        isDeleted
          ? "border-red-400/30 bg-red-400/5 opacity-60"
          : "border-[var(--color-border)] bg-[var(--color-surface)]"
      }`}
      style={{ width: nodeWidth || 160 }}
      data-hovered={hovered || undefined}
      onClick={() => !isDeleted && router.push(`/agents/${label}`)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <Handle type="target" position={Position.Top} className={hasIncoming ? handleVisible : handleHidden} />

      <div className="flex items-center gap-2">
        <span
          className={`inline-block h-2 w-2 shrink-0 rounded-full ${statusDotClass(status)}`}
        />
        <span className={`text-sm font-medium truncate ${isDeleted ? "text-red-400/70 line-through" : "text-[var(--color-text)]"}`}>
          {label}
        </span>
      </div>

      <p className="mt-1 text-[10px] text-[var(--color-text-secondary)]">
        {isDeleted ? "deleted" : model || "claude"}
      </p>

      <Handle type="source" position={Position.Bottom} className={hasOutgoing ? handleVisible : handleHidden} />

      {hovered && !isDeleted && (
        <div className="absolute left-full top-0 ml-3 z-50 min-w-52 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-raised)] p-3 shadow-[0_8px_32px_rgba(var(--shadow-color),var(--shadow-strength))] pointer-events-none animate-[tooltipIn_0.15s_ease-out]">
          <TooltipHeader
            icon={<span className={`inline-block h-2.5 w-2.5 rounded-full ${statusDotClass(status)}`} />}
            title={label}
            subtitle={namespace}
          />
          <div className="space-y-1.5">
            <TooltipRow label="Status" value={status} color={statusColorMap[status]} />
            <TooltipRow label="Task" value={taskStatus || "Idle"} color={taskStatus ? statusColorMap[taskStatus] : undefined} />
            <TooltipRow label="Model" value={model} />
            <TooltipRow label="Lifecycle" value={lifecycle || "Default"} />
            <TooltipRow label="Total Cost" value={totalCostUSD ? `$${totalCostUSD}` : undefined} color={totalCostUSD ? "var(--color-brand-blue)" : undefined} />
          </div>
        </div>
      )}
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
  namespace?: string;
  manager?: string;
  totalCostUSD?: string;
  hasIncoming?: boolean;
  hasOutgoing?: boolean;
  nodeWidth?: number;
};

function OfficeNodeComponent({ data }: NodeProps) {
  const router = useRouter();
  const { label, phase, agentCount, namespace, manager, totalCostUSD, hasIncoming, hasOutgoing, nodeWidth } = data as unknown as OfficeNodeData;
  const [hovered, setHovered] = useState(false);

  return (
    <div
      className="relative cursor-pointer rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-raised)] px-5 py-3.5 transition-shadow hover:shadow-[0_4px_16px_rgba(var(--shadow-color),var(--shadow-strength)),inset_0_1px_0_var(--color-border-light)]"
      style={{ width: nodeWidth || 220 }}
      data-hovered={hovered || undefined}
      onClick={() => router.push(`/offices/${label}`)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <Handle type="target" position={Position.Top} className={hasIncoming ? handleVisible : handleHidden} />

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

      <Handle type="source" position={Position.Bottom} className={hasOutgoing ? handleVisible : handleHidden} />

      {hovered && (
        <div className="absolute left-full top-0 ml-3 z-50 min-w-52 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-raised)] p-3 shadow-[0_8px_32px_rgba(var(--shadow-color),var(--shadow-strength))] pointer-events-none animate-[tooltipIn_0.15s_ease-out]">
          <TooltipHeader
            icon={<Building2 className="h-3.5 w-3.5 text-[var(--color-brand-blue)]" />}
            title={label}
            subtitle={namespace}
          />
          <div className="space-y-1.5">
            <TooltipRow label="Phase" value={phase} color={statusColorMap[phase]} />
            <TooltipRow label="Manager" value={manager} />
            <TooltipRow label="Agents" value={agentCount} />
            <TooltipRow label="Total Cost" value={totalCostUSD ? `$${totalCostUSD}` : undefined} color={totalCostUSD ? "var(--color-brand-blue)" : undefined} />
          </div>
        </div>
      )}
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
  namespace?: string;
  agentName?: string;
  totalCostUSD?: string;
  runCount?: number;
  hasIncoming?: boolean;
  hasOutgoing?: boolean;
  nodeWidth?: number;
};

function ScheduleNodeComponent({ data }: NodeProps) {
  const router = useRouter();
  const { label, cron, phase, namespace, agentName, totalCostUSD, runCount, hasIncoming, hasOutgoing, nodeWidth } = data as unknown as ScheduleNodeData;
  const [hovered, setHovered] = useState(false);

  return (
    <div
      className="relative cursor-pointer rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-3 shadow-md transition-shadow hover:shadow-lg"
      style={{ width: nodeWidth || 170 }}
      data-hovered={hovered || undefined}
      onClick={() => router.push(`/schedules/${label}`)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <Handle type="target" position={Position.Top} className={hasIncoming ? handleVisible : handleHidden} />

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

      <Handle type="source" position={Position.Bottom} className={hasOutgoing ? handleVisible : handleHidden} />

      {hovered && (
        <div className="absolute left-full top-0 ml-3 z-50 min-w-52 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-raised)] p-3 shadow-[0_8px_32px_rgba(var(--shadow-color),var(--shadow-strength))] pointer-events-none animate-[tooltipIn_0.15s_ease-out]">
          <TooltipHeader
            icon={<Clock className="h-3.5 w-3.5 text-cyan-400" />}
            title={label}
            subtitle={namespace}
          />
          <div className="space-y-1.5">
            <TooltipRow label="Phase" value={phase} color={statusColorMap[phase]} />
            <TooltipRow label="Schedule" value={cron} />
            <TooltipRow label="Agent" value={agentName} />
            <TooltipRow label="Runs" value={runCount} />
            <TooltipRow label="Total Cost" value={totalCostUSD ? `$${totalCostUSD}` : undefined} color={totalCostUSD ? "var(--color-brand-blue)" : undefined} />
          </div>
        </div>
      )}
    </div>
  );
}

export const ScheduleNode = memo(ScheduleNodeComponent);
