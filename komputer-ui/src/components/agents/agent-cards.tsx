"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { Bot, Trash2, Zap, Moon, Skull, Clock, CheckCircle2, Check } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { Button } from "@/components/kit/button";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { formatCost, formatRelativeTime } from "@/lib/utils";
import type { AgentResponse } from "@/lib/types";

export const agentKey = (a: { name: string; namespace: string }) => `${a.namespace}/${a.name}`;

type AgentCardsProps = {
  agents: AgentResponse[];
  onDelete: (name: string, namespace: string) => void;
  selected?: Set<string>;
  onToggleSelect?: (key: string) => void;
};

const statusConfig: Record<string, { color: string; icon: typeof Bot; pulse?: boolean }> = {
  Running:   { color: "#34D399", icon: Zap, pulse: true },
  Sleeping:  { color: "#FBBF24", icon: Moon },
  Pending:   { color: "#FBBF24", icon: Clock },
  Failed:    { color: "#F87171", icon: Skull },
  Succeeded: { color: "#34D399", icon: CheckCircle2 },
};

const defaultStatus = { color: "#8899A6", icon: Bot };

export function AgentCards({ agents, onDelete, selected, onToggleSelect }: AgentCardsProps) {
  const selectionMode = !!selected && selected.size > 0;
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-2.5">
      <AnimatePresence>
        {agents.map((agent, i) => {
          const cfg = statusConfig[agent.status] ?? defaultStatus;
          const StatusIcon = cfg.icon;
          const isActive = agent.taskStatus === "InProgress";
          const key = agentKey(agent);
          const isSelected = !!selected?.has(key);

          const handleCardClick = (e: React.MouseEvent) => {
            if (selectionMode && onToggleSelect) {
              e.preventDefault();
              onToggleSelect(key);
            }
          };
          const handleCheckboxClick = (e: React.MouseEvent) => {
            e.preventDefault();
            e.stopPropagation();
            onToggleSelect?.(key);
          };

          return (
            <motion.div
              key={agent.name}
              initial={{ opacity: 0, y: 12, scale: 0.97 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, scale: 0.97 }}
              transition={{ duration: 0.25, delay: i * 0.04 }}
            >
              <Link href={`/agents/${agent.name}?namespace=${agent.namespace}`} className="block group" onClick={handleCardClick}>
                <div
                  className={`relative rounded-[var(--radius-md)] border bg-[var(--color-surface)] ${
                    isSelected
                      ? "border-[var(--color-border)]"
                      : "border-[var(--color-border)] transition-[border-color,box-shadow] duration-150 group-hover:border-[var(--color-border-hover)] group-hover:shadow-[0_0_20px_rgba(63,133,217,0.06),0_0_40px_rgba(139,92,246,0.04)]"
                  }`}
                  style={isSelected ? { boxShadow: "0 0 20px rgba(63,133,217,0.15)" } : undefined}
                >
                  <SelectionBorder active={isSelected} />


                  <div className="p-3 flex flex-col h-full">
                    {/* Top row: icon/checkbox + name + delete + dot */}
                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        onClick={handleCheckboxClick}
                        className={`relative flex items-center justify-center w-7 h-7 rounded-md shrink-0 cursor-pointer ${
                          isSelected
                            ? "bg-[var(--color-brand-blue)]"
                            : `${selectionMode ? "" : "group-hover:ring-1 group-hover:ring-[var(--color-border-hover)]"}`
                        }`}
                        style={!isSelected ? { backgroundColor: `${cfg.color}15` } : undefined}
                        aria-label={isSelected ? "Deselect agent" : "Select agent"}
                      >
                        {isSelected ? (
                          <Check
                            className="w-3.5 h-3.5 text-white"
                            strokeWidth={3}
                            style={{ animation: "agent-check-pop 180ms ease-out" }}
                          />
                        ) : (
                          <span className="relative inline-flex">
                            <StatusIcon
                              className={`w-3.5 h-3.5 ${onToggleSelect ? "group-hover:opacity-0" : ""}`}
                              style={{ color: cfg.color }}
                            />
                            {onToggleSelect && (
                              <span className="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100">
                                <span className="w-3.5 h-3.5 rounded border border-[var(--color-text-secondary)]" />
                              </span>
                            )}
                          </span>
                        )}
                      </button>
                      <span className="text-[13px] font-semibold text-[var(--color-text)] truncate leading-tight flex-1 min-w-0">
                        {agent.name}
                      </span>
                      <div className="flex items-center gap-1.5 shrink-0">
                        <div onClick={(e) => { e.stopPropagation(); e.preventDefault(); }} className="opacity-0 group-hover:opacity-100 transition-opacity">
                          <ConfirmDialog
                            title={`Delete ${agent.name}?`}
                            description="This will permanently delete this agent and its workspace."
                            onConfirm={() => onDelete(agent.name, agent.namespace)}
                            trigger={
                              <Button variant="ghost" size="icon" className="h-5 w-5 p-0">
                                <Trash2 className="w-2.5 h-2.5 text-[var(--color-text-secondary)] hover:text-red-400 transition-colors" />
                              </Button>
                            }
                          />
                        </div>
                        <span
                          className="block w-2 h-2 rounded-full"
                          style={{
                            backgroundColor: cfg.color,
                            boxShadow: (cfg as typeof defaultStatus & { pulse?: boolean }).pulse && isActive
                              ? `0 0 6px ${cfg.color}80`
                              : undefined,
                          }}
                        />
                      </div>
                    </div>

                    {/* Bottom: fields */}
                    <div className="mt-4 space-y-1">
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">status</span>
                        <span className="text-[11px] font-medium text-[var(--color-text-secondary)]">
                          {agent.status}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">model</span>
                        <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text-secondary)] truncate ml-2">
                          {agent.model?.replace("claude-", "")}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">cost</span>
                        <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text)]">
                          {formatCost(agent.totalCostUSD)}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">age</span>
                        <span className="text-[11px] text-[var(--color-text-secondary)]">
                          {formatRelativeTime(agent.createdAt)}
                        </span>
                      </div>
                    </div>
                  </div>

                </div>
              </Link>
            </motion.div>
          );
        })}
      </AnimatePresence>
    </div>
  );
}

// Animated selection border. Renders an SVG <rect> matching the parent's
// pixel size so rounded corners stay perfectly circular (no aspect-ratio
// stretching). Animates the stroke from 0% → 100% on select and 100% → 0%
// on deselect. Stays mounted briefly during the deselect to play the exit.
function SelectionBorder({ active }: { active: boolean }) {
  const [show, setShow] = useState(active);
  const [drawing, setDrawing] = useState<"in" | "out">(active ? "in" : "out");
  const [animKey, setAnimKey] = useState(0); // forces rect remount on every transition
  const [size, setSize] = useState<{ w: number; h: number } | null>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  // Measure the parent on mount and on resize.
  useEffect(() => {
    const el = wrapperRef.current?.parentElement;
    if (!el) return;
    const update = () => setSize({ w: el.clientWidth, h: el.clientHeight });
    update();
    const ro = new ResizeObserver(update);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  useEffect(() => {
    if (active) {
      setShow(true);
      setDrawing("in");
      setAnimKey((k) => k + 1);
    } else if (show) {
      setDrawing("out");
      setAnimKey((k) => k + 1);
      const t = setTimeout(() => setShow(false), 280);
      return () => clearTimeout(t);
    }
    // intentionally not depending on `show` to avoid double-fires
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active]);

  if (!show || !size) {
    // Keep ref mounted so we can measure the parent.
    return <div ref={wrapperRef} className="hidden" aria-hidden />;
  }

  // Draw rect at +1px outside so it overlaps the existing 1px border cleanly.
  const w = size.w + 2;
  const h = size.h + 2;
  const r = 11;
  // Build a rounded-rect path so we can compute the exact perimeter for the
  // stroke-dasharray/offset animation. (pathLength on <rect> isn't reliably
  // supported across browsers for animated dashoffset.)
  const innerW = w - 3;
  const innerH = h - 3;
  const x0 = 1.5;
  const y0 = 1.5;
  const path = [
    `M ${x0 + r} ${y0}`,
    `H ${x0 + innerW - r}`,
    `A ${r} ${r} 0 0 1 ${x0 + innerW} ${y0 + r}`,
    `V ${y0 + innerH - r}`,
    `A ${r} ${r} 0 0 1 ${x0 + innerW - r} ${y0 + innerH}`,
    `H ${x0 + r}`,
    `A ${r} ${r} 0 0 1 ${x0} ${y0 + innerH - r}`,
    `V ${y0 + r}`,
    `A ${r} ${r} 0 0 1 ${x0 + r} ${y0}`,
    "Z",
  ].join(" ");
  // Approximate perimeter: 2(w + h) - 8r + 2πr.
  const perimeter = 2 * (innerW + innerH) - 8 * r + 2 * Math.PI * r;

  return (
    <>
      <div ref={wrapperRef} className="hidden" aria-hidden />
      <svg
        className="pointer-events-none absolute"
        style={{ left: -1, top: -1, width: w, height: h }}
        width={w}
        height={h}
        viewBox={`0 0 ${w} ${h}`}
      >
        <path
          key={`${drawing}-${animKey}`}
          d={path}
          fill="none"
          stroke="var(--color-brand-blue)"
          strokeWidth={3}
          strokeLinejoin="round"
          strokeLinecap="round"
          style={{
            strokeDasharray: perimeter,
            strokeDashoffset: drawing === "in" ? perimeter : 0,
            animation: drawing === "in"
              ? `agent-border-draw-in 320ms ease-out forwards`
              : `agent-border-draw-out 260ms ease-in forwards`,
            ["--perimeter" as string]: `${perimeter}px`,
          }}
        />
      </svg>
    </>
  );
}
