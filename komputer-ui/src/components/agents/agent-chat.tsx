"use client";

import React, { useState, useEffect, useLayoutEffect, useRef, useCallback, useMemo } from "react";
import Link from "next/link";
import { motion, AnimatePresence, useAnimation } from "framer-motion";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { AgentEvent } from "@/lib/types";
import { createAgent, cancelAgent, patchAgent } from "@/lib/api";
import { getConfig } from "@/lib/config";
import { CostBadge } from "@/components/shared/cost-badge";
import { CopyButton } from "@/components/shared/copy-button";
import { cn } from "@/lib/utils";
import {
  ChevronRight,
  ArrowUp,
  ArrowDown,
  Square,
  Terminal,
  FileText,
  Globe,
  Wrench,
  Settings2,
  Moon,
  Trash2,
  Play,
  Sparkles,
} from "lucide-react";

type AgentChatProps = {
  agentName: string;
  agentNamespace?: string;
  agentStatus: string;
  agentLifecycle?: string;
  agentContextWindow?: number;
  events: AgentEvent[];
  taskStatus?: string;
  initialPending?: string;
  hasMoreEvents?: boolean;
  loadingOlder?: boolean;
  onLoadOlder?: () => void;
  scrollContainerRef?: React.RefObject<HTMLElement | null>;
  scrollSnapshotRef?: React.RefObject<number | null>;
  highlightTaskFrom?: string;
  highlightTaskTo?: string;
  hasNewerEvents?: boolean;
  loadingNewer?: boolean;
  onLoadNewer?: () => void;
};

// --- Message types derived from events ---

type TokenUsage = {
  input_tokens?: number;
  output_tokens?: number;
  cache_read_input_tokens?: number;
  cache_creation_input_tokens?: number;
};


type ChatMessage =
  | { kind: "user"; text: string; timestamp: string }
  | { kind: "text"; text: string; timestamp: string; usage?: TokenUsage }
  | { kind: "thinking"; text: string; timestamp: string; usage?: TokenUsage }
  | {
      kind: "tool";
      toolName: string;
      description?: string;
      command?: string;
      input?: unknown;
      output?: unknown;
      timestamp: string;
    }
  | {
      kind: "completed";
      costUSD?: string;
      duration?: string;
      turns?: number;
      inputTokens?: number;
      contextTokens?: number;
      outputTokens?: number;
      cacheReadTokens?: number;
      cacheCreationTokens?: number;
      timestamp: string;
    }
  | { kind: "error"; message: string; timestamp: string }
  | { kind: "cancelled"; timestamp: string }
  | { kind: "compaction"; timestamp: string };

export function eventsToChatMessages(events: AgentEvent[]): ChatMessage[] {
  const messages: ChatMessage[] = [];
  for (const event of events) {
    switch (event.type) {
      case "task_started": {
        const instructions =
          (event.payload.instructions ?? event.payload.message ?? "").trim();
        if (instructions) {
          messages.push({
            kind: "user",
            text: instructions,
            timestamp: event.timestamp,
          });
        }
        break;
      }
      case "user_message": {
        const content = (event.payload.content ?? "").trim();
        if (content) {
          messages.push({
            kind: "user",
            text: content,
            timestamp: event.timestamp,
          });
        }
        break;
      }
      case "text": {
        const content = event.payload.content ?? event.payload.text ?? "";
        if (content) {
          messages.push({
            kind: "text",
            text: content,
            timestamp: event.timestamp,
            usage: event.payload.usage as TokenUsage | undefined,
          });
        }
        break;
      }
      case "thinking": {
        const content = event.payload.content ?? event.payload.text ?? "";
        if (content) {
          messages.push({
            kind: "thinking",
            text: content,
            timestamp: event.timestamp,
            usage: event.payload.usage as TokenUsage | undefined,
          });
        }
        break;
      }
      case "tool_result": {
        let toolName = event.payload.tool ?? event.payload.name ?? "tool";
        const inp = event.payload.input;
        let description: string | undefined;
        let inputSummary: string | undefined;
        let output: unknown;
        // Strip mcp__<connector>__ prefix and use connector name as description
        const mcpMatch = toolName.match(/^mcp__([^_]+(?:_[^_]+)*)__(.+)$/);
        if (mcpMatch) {
          description = mcpMatch[1].replace(/_/g, "-");
          toolName = mcpMatch[2];
        }
        if (toolName === "Skill") {
          description = inp?.skill;
          inputSummary = inp?.args ? String(inp.args) : undefined;
          // Don't show the raw skill file content as output
        } else {
          description = description ?? inp?.description ?? event.payload.description;
          if (inp?.command ?? inp?.cmd) {
            inputSummary = inp.command ?? inp.cmd;
          } else if (inp && typeof inp === "object") {
            const parts = Object.entries(inp)
              .filter(([k]) => k !== "description")
              .map(([k, v]) => `${k}=${typeof v === "string" ? v : JSON.stringify(v)}`);
            if (parts.length > 0) inputSummary = parts.join(" ");
          }
          output = event.payload.output ?? event.payload.content;
        }
        messages.push({
          kind: "tool",
          toolName,
          description,
          command: inputSummary,
          input: toolName === "Skill" ? undefined : inp,
          output,
          timestamp: event.timestamp,
        });
        break;
      }
      case "task_completed": {
        const costRaw = event.payload.costUSD ?? event.payload.cost_usd;
        const durationMs = event.payload.duration ?? event.payload.duration_ms;
        const duration = typeof durationMs === "number" ? `${(durationMs / 1000).toFixed(1)}s` : durationMs;
        const cost = typeof costRaw === "number" ? costRaw.toFixed(4) : costRaw;
        const usage = event.payload.usage as TokenUsage | undefined;
        // last_usage is from the final API call — represents actual context size (for the context bar).
        const lastUsage = (event.payload.last_usage as TokenUsage | undefined) ?? usage;
        messages.push({
          kind: "completed",
          costUSD: cost,
          duration,
          turns: event.payload.turns ?? event.payload.num_turns,
          inputTokens: usage
            ? (usage.input_tokens ?? 0) + (usage.cache_read_input_tokens ?? 0) + (usage.cache_creation_input_tokens ?? 0)
            : undefined,
          contextTokens: lastUsage
            ? (lastUsage.input_tokens ?? 0) + (lastUsage.cache_read_input_tokens ?? 0) + (lastUsage.cache_creation_input_tokens ?? 0)
            : undefined,
          outputTokens: usage?.output_tokens,
          cacheReadTokens: usage?.cache_read_input_tokens,
          cacheCreationTokens: usage?.cache_creation_input_tokens,
          timestamp: event.timestamp,
        });
        break;
      }
      case "task_cancelled":
        messages.push({ kind: "cancelled", timestamp: event.timestamp });
        break;
      case "compaction":
        messages.push({ kind: "compaction", timestamp: event.timestamp });
        break;
      case "error":
        messages.push({
          kind: "error",
          message:
            event.payload.message ?? event.payload.error ?? "Unknown error",
          timestamp: event.timestamp,
        });
        break;
    }
  }
  return messages;
}

function formatTimestamp(ts: string): string {
  const d = new Date(ts);
  const now = new Date();
  const time = d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  if (d.toDateString() === now.toDateString()) return time;
  return `${d.toLocaleDateString([], { year: "numeric", month: "2-digit", day: "2-digit" })} ${time}`;
}

function formatTokenCount(n: number): string {
  if (n >= 1_000_000) {
    const v = n / 1_000_000;
    return `${Number.isInteger(v) ? v : v.toFixed(1)}m`;
  }
  if (n >= 1000) {
    const v = n / 1000;
    return `${Number.isInteger(v) ? v : v.toFixed(1)}k`;
  }
  return String(n);
}

function TokenBadge({ usage }: { usage?: TokenUsage }) {
  if (!usage) return null;
  const total = (usage.input_tokens ?? 0) + (usage.output_tokens ?? 0);
  if (total === 0) return null;
  return (
    <span className="text-[10px] text-[var(--color-text-muted)] tabular-nums">
      {formatTokenCount(total)} tokens
    </span>
  );
}

function ContextBar({ inputTokens, contextWindow }: { inputTokens?: number; contextWindow: number }) {
  if (inputTokens == null || inputTokens === 0) return null;
  const pct = Math.min((inputTokens / contextWindow) * 100, 100);
  // Hold the bar to the dark-theme palette in both modes — the brand blue
  // here doubles as a fill on the muted track and stays legible without
  // re-tuning per theme.
  const color =
    pct >= 90 ? "#F87171" :
    pct >= 70 ? "#FBBF24" :
    "#3f85d9";
  return (
    <div className="group relative h-[12px] cursor-default">
      {/* Bar sits 8px above the border edge */}
      <div className="absolute bottom-0 left-0 right-0 h-[5px] bg-[var(--color-border)] transition-[height] duration-150 group-hover:h-[12px]">
        <div
          className="h-full transition-[width] duration-150 ease-out"
          style={{ width: `${pct}%`, backgroundColor: color } as React.CSSProperties}
        />
      </div>
      <div className="pointer-events-none absolute bottom-full left-1/2 mb-4 -translate-x-1/2 z-20 whitespace-nowrap rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 py-1.5 text-[11px] text-[var(--color-text-secondary)] opacity-0 shadow-lg transition-opacity duration-150 group-hover:opacity-100">
        <span className="text-[var(--color-text)]">Context window</span>
        {" · "}
        <span className="font-mono tabular-nums" style={{ color }}>{formatTokenCount(inputTokens)}</span>
        <span className="text-[var(--color-text)]"> / {formatTokenCount(contextWindow)} tokens</span>
      </div>
    </div>
  );
}

function getToolIcon(name: string) {
  const lower = name.toLowerCase();
  if (lower === "skill")
    return <Sparkles className="size-3.5 shrink-0" />;
  if (lower.includes("bash") || lower.includes("shell"))
    return <Terminal className="size-3.5 shrink-0" />;
  if (lower.includes("read") || lower.includes("write") || lower.includes("edit"))
    return <FileText className="size-3.5 shrink-0" />;
  if (lower.includes("web") || lower.includes("fetch") || lower.includes("curl"))
    return <Globe className="size-3.5 shrink-0" />;
  return <Wrench className="size-3.5 shrink-0" />;
}

// --- Sub-components ---

function UserBubble({ text, timestamp }: { text: string; timestamp: string }) {
  return (
    <motion.div
      className="group/msg flex justify-end"
      initial={{ opacity: 0, x: 12 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.25, ease: "easeOut" }}
    >
      <div className="relative max-w-[80%] rounded-xl rounded-br-sm bg-[var(--color-surface)] px-4 py-2.5 transition-colors duration-150 hover:bg-[var(--color-surface-hover)]">
        <div className="absolute -left-8 top-1 opacity-0 group-hover/msg:opacity-100 transition-opacity duration-200">
          <CopyButton text={text} />
        </div>
        <div className="prose-chat text-sm text-[var(--color-text)] break-words">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{text}</ReactMarkdown>
        </div>
        <p className="mt-1 text-right text-[10px] text-[var(--color-text-secondary)]">
          {formatTimestamp(timestamp)}
        </p>
      </div>
    </motion.div>
  );
}

// Replace /workspace/files/ paths in agent text with clickable download links.
function linkifyFiles(text: string, agentName: string, namespace: string): string {
  return text.replace(
    /(?:`)?\/workspace\/files\/([\w.\-\/]+)(?:`)?/g,
    (_, filePath) => `[📥 ${filePath}](${getConfig().apiUrl}/api/v1/agents/${agentName}/download/${filePath}?namespace=${namespace})`
  );
}

function AgentBubble({ text, timestamp, usage, agentName, namespace }: { text: string; timestamp: string; usage?: TokenUsage; agentName?: string; namespace?: string }) {
  const displayText = agentName ? linkifyFiles(text, agentName, namespace || "default") : text;
  return (
    <motion.div
      className="group/msg flex justify-start"
      initial={{ opacity: 0, x: -12 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.25, ease: "easeOut" }}
    >
      <div className="relative max-w-[80%] rounded-xl px-4 py-2.5 transition-colors duration-150 hover:bg-[var(--color-surface)]">
        <div className="prose-chat text-sm text-[var(--color-text)]">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{displayText}</ReactMarkdown>
        </div>
        <div className="mt-1 flex items-center gap-1.5">
          <span className="text-[10px] text-[var(--color-text-secondary)]">
            {formatTimestamp(timestamp)}
          </span>
          {usage && (
            <>
              <span className="text-[10px] text-[var(--color-text-muted)]">·</span>
              <TokenBadge usage={usage} />
            </>
          )}
          <span className="opacity-0 group-hover/msg:opacity-100 transition-opacity duration-200">
            <CopyButton text={text} />
          </span>
        </div>
      </div>
    </motion.div>
  );
}

function ThinkingBubble({ text }: { text: string }) {
  return (
    <motion.div
      className="flex justify-start"
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 0.7, y: 0 }}
      exit={{ opacity: 0, y: -4 }}
      transition={{ duration: 0.3, ease: "easeOut" }}
    >
      <div className="max-w-[80%]">
        <p className="text-sm italic text-[var(--color-text-secondary)]">{text}</p>
      </div>
    </motion.div>
  );
}

function EditDiff({ oldStr, newStr }: { oldStr: string; newStr: string }) {
  const oldLines = oldStr.split("\n");
  const newLines = newStr.split("\n");
  const maxLines = Math.max(oldLines.length, newLines.length);

  // Build per-line diff status
  const leftLines: { text: string; status: "removed" | "changed" | "same" }[] = [];
  const rightLines: { text: string; status: "added" | "changed" | "same" }[] = [];

  for (let i = 0; i < maxLines; i++) {
    const o = i < oldLines.length ? oldLines[i] : undefined;
    const n = i < newLines.length ? newLines[i] : undefined;
    if (o !== undefined && n !== undefined) {
      if (o === n) {
        leftLines.push({ text: o, status: "same" });
        rightLines.push({ text: n, status: "same" });
      } else {
        leftLines.push({ text: o, status: "changed" });
        rightLines.push({ text: n, status: "changed" });
      }
    } else if (o !== undefined) {
      leftLines.push({ text: o, status: "removed" });
      rightLines.push({ text: "", status: "same" });
    } else if (n !== undefined) {
      leftLines.push({ text: "", status: "same" });
      rightLines.push({ text: n, status: "added" });
    }
  }

  const renderLine = (line: { text: string; status: string }, i: number, side: "old" | "new") => {
    const isChanged = line.status === "removed" || line.status === "changed" || line.status === "added";
    const bg = !isChanged ? "" : side === "old" ? "bg-red-500/8" : "bg-green-500/8";
    const numBg = !isChanged ? "" : side === "old" ? "bg-red-500/5" : "bg-green-500/5";
    const textColor = !isChanged ? "text-[var(--color-text-muted)]" : side === "old" ? "text-red-300/80" : "text-green-300/80";
    const marker = !isChanged ? " " : side === "old" ? "-" : "+";
    const markerColor = !isChanged ? "text-transparent" : side === "old" ? "text-red-400/60" : "text-green-400/60";

    return (
      <div key={i} className={`flex border-b border-[var(--color-border)]/30 last:border-b-0 ${bg}`}>
        <span className={`select-none w-7 shrink-0 text-right pr-1.5 py-0.5 text-[10px] text-[var(--color-text-muted)]/50 ${numBg}`}>{line.text !== "" ? i + 1 : ""}</span>
        <span className={`select-none w-5 shrink-0 text-center py-0.5 ${markerColor}`}>{marker}</span>
        <span className={`py-0.5 pr-2 whitespace-pre-wrap break-words min-w-0 ${textColor}`}>{line.text || " "}</span>
      </div>
    );
  };

  return (
    <div className="rounded-md border border-[var(--color-border)] overflow-hidden">
      <div className="grid grid-cols-2 divide-x divide-[var(--color-border)] font-mono text-xs">
        <div className="min-w-0">
          {leftLines.map((line, i) => renderLine(line, i, "old"))}
        </div>
        <div className="min-w-0">
          {rightLines.map((line, i) => renderLine(line, i, "new"))}
        </div>
      </div>
    </div>
  );
}

function ToolCard({
  toolName,
  description,
  command,
  input,
  output,
}: {
  toolName: string;
  description?: string;
  command?: string;
  input?: unknown;
  output?: unknown;
}) {
  const [open, setOpen] = useState(false);

  return (
    <div className="overflow-hidden rounded-md border border-[var(--color-border)] bg-[var(--color-surface)] transition-colors duration-150 hover:border-[var(--color-border-hover)] hover:bg-[var(--color-surface-hover)]">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left cursor-pointer"
      >
        <ChevronRight
          className={cn(
            "size-3.5 shrink-0 text-[var(--color-text-secondary)] transition-transform duration-200",
            open && "rotate-90"
          )}
        />
        {getToolIcon(toolName)}
        {description && (
          <span className="shrink-0 text-sm font-semibold text-[var(--color-text)]">
            {description}
          </span>
        )}
        <span className="shrink-0 text-sm font-semibold text-[var(--color-text)]">
          {toolName}
        </span>
        {command && (
          <code className="min-w-0 truncate text-xs font-mono text-[var(--color-text-muted)]">
            {command}
          </code>
        )}
      </button>
      <AnimatePresence initial={false}>
        {open && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
            className="overflow-hidden"
          >
            <div className="border-t border-[var(--color-border)] px-3 py-2">
              {toolName === "Edit" && input != null && typeof input === "object" && (input as Record<string, unknown>).old_string != null ? (
                <EditDiff
                  oldStr={String((input as Record<string, unknown>).old_string ?? "")}
                  newStr={String((input as Record<string, unknown>).new_string ?? "")}
                />
              ) : (
                <>
                  {input != null && (
                    <div className="mb-2">
                      <p className="mb-1 text-[10px] font-medium uppercase text-[var(--color-text-secondary)]">
                        Input
                      </p>
                      <pre className="whitespace-pre-wrap break-words rounded bg-[var(--color-bg)] p-2 font-mono text-xs text-[var(--color-text)]" style={{ overflowWrap: "anywhere" }}>
                        {typeof input === "string"
                          ? input
                          : JSON.stringify(input, null, 2)}
                      </pre>
                    </div>
                  )}
                  {output != null && (
                    <div>
                      <p className="mb-1 text-[10px] font-medium uppercase text-[var(--color-text-secondary)]">
                        Output
                      </p>
                      <pre className="max-h-60 overflow-y-auto whitespace-pre-wrap break-words rounded bg-[var(--color-bg)] p-2 font-mono text-xs text-[var(--color-text)]" style={{ overflowWrap: "anywhere" }}>
                        {typeof output === "string"
                          ? output
                          : JSON.stringify(output, null, 2)}
                      </pre>
                    </div>
                  )}
                </>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

function SkillCard({ skillName, args }: { skillName: string; args?: string }) {
  return (
    <Link
      href={`/skills/${encodeURIComponent(skillName)}`}
      className="flex items-center gap-2 px-3 py-2 rounded-md border border-[var(--color-border)] bg-[var(--color-surface)] transition-colors duration-150 hover:border-[var(--color-border-hover)] hover:bg-[var(--color-surface-hover)] cursor-pointer"
    >
      <Sparkles className="size-3.5 shrink-0 text-[var(--color-brand-violet)]" />
      <span className="shrink-0 text-sm font-semibold text-[var(--color-text)]">
        Skill
      </span>
      <span className="shrink-0 text-sm text-[var(--color-text-secondary)]">
        {skillName}
      </span>
      {args && (
        <code className="min-w-0 truncate text-xs font-mono text-[var(--color-text-muted)]">
          {args}
        </code>
      )}
    </Link>
  );
}

function CompletedDivider({
  costUSD,
  duration,
  turns,
  inputTokens,
  outputTokens,
  cacheReadTokens,
  cacheCreationTokens,
}: {
  costUSD?: string;
  duration?: string;
  turns?: number;
  inputTokens?: number;
  outputTokens?: number;
  cacheReadTokens?: number;
  cacheCreationTokens?: number;
}) {
  const total = (inputTokens ?? 0) + (outputTokens ?? 0);
  const newInput = total - (cacheReadTokens ?? 0) - (cacheCreationTokens ?? 0) - (outputTokens ?? 0);
  const hasBreakdown = total > 0 && (cacheReadTokens || cacheCreationTokens);
  const breakdownControls = useAnimation();
  useEffect(() => { breakdownControls.set({ opacity: 0, x: -8 }); }, []);

  return (
    <motion.div
      className="py-2"
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.4, ease: "easeOut" }}
    >
      {/* Top line with label */}
      <motion.div
        className="flex items-center gap-3"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.3, delay: 0.1 }}
      >
        <motion.div
          className="flex-1 border-t border-green-400/20"
          initial={{ scaleX: 0 }}
          animate={{ scaleX: 1 }}
          transition={{ duration: 0.5, delay: 0.2, ease: "easeOut" }}
          style={{ transformOrigin: "right" }}
        />
        <motion.span
          className="text-xs font-medium text-green-400"
          initial={{ opacity: 0, y: 4 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.15 }}
        >
          Task completed
        </motion.span>
        <motion.div
          className="flex-1 border-t border-green-400/20"
          initial={{ scaleX: 0 }}
          animate={{ scaleX: 1 }}
          transition={{ duration: 0.5, delay: 0.2, ease: "easeOut" }}
          style={{ transformOrigin: "left" }}
        />
      </motion.div>

      {/* Stats row */}
      {(costUSD || duration || turns != null) && (
        <motion.div
          className="mt-1.5 flex items-center justify-center gap-3"
          initial={{ opacity: 0, y: 6 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.4 }}
        >
          {costUSD && (
            <span className="rounded-full bg-[var(--color-surface-raised)] px-2.5 py-0.5 font-semibold [&>span]:text-[var(--color-text)]">
              <CostBadge cost={costUSD} />
            </span>
          )}
          {duration && (
            <span className="text-xs text-[var(--color-text-secondary)]">
              {duration}
            </span>
          )}
          {turns != null && (
            <span className="text-xs text-[var(--color-text-secondary)]">
              {turns} turn{turns !== 1 ? "s" : ""}
            </span>
          )}
          {total > 0 && (
            <span
              className="relative flex items-center text-xs tabular-nums text-[var(--color-text-secondary)]"
              onMouseEnter={() => hasBreakdown && breakdownControls.start({ opacity: 1, x: 0, transition: { duration: 0.2, ease: "easeOut" } })}
              onMouseLeave={() => hasBreakdown && breakdownControls.start({ opacity: 0, x: -8, transition: { duration: 0.15, ease: "easeIn" } })}
            >
              {formatTokenCount(total)} tokens
              {hasBreakdown && (
                <motion.span
                  className="absolute left-full ml-2 flex items-center gap-3 text-[11px] text-[var(--color-text-muted)] whitespace-nowrap pointer-events-none"
                  initial={{ opacity: 0, x: -8 }}
                  animate={breakdownControls}
                >
                  <span className="pl-1 border-l border-[var(--color-border)]">in <span className="text-[var(--color-text-secondary)]">{formatTokenCount(newInput)}</span></span>
                  <span>out <span className="text-[var(--color-text-secondary)]">{formatTokenCount(outputTokens ?? 0)}</span></span>
                  {(cacheReadTokens ?? 0) > 0 && <span>cache read <span className="text-[var(--color-text-secondary)]">{formatTokenCount(cacheReadTokens!)}</span></span>}
                  {(cacheCreationTokens ?? 0) > 0 && <span>cache write <span className="text-[var(--color-text-secondary)]">{formatTokenCount(cacheCreationTokens!)}</span></span>}
                </motion.span>
              )}
            </span>
          )}
        </motion.div>
      )}
    </motion.div>
  );
}

function CancelledDivider() {
  return (
    <motion.div
      className="py-2"
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.4, ease: "easeOut" }}
    >
      <motion.div
        className="flex items-center gap-3"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.3, delay: 0.1 }}
      >
        <motion.div
          className="flex-1 border-t border-amber-400/20"
          initial={{ scaleX: 0 }}
          animate={{ scaleX: 1 }}
          transition={{ duration: 0.5, delay: 0.2, ease: "easeOut" }}
          style={{ transformOrigin: "right" }}
        />
        <motion.span
          className="text-xs font-medium text-amber-400"
          initial={{ opacity: 0, y: 4 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.15 }}
        >
          Task cancelled
        </motion.span>
        <motion.div
          className="flex-1 border-t border-amber-400/20"
          initial={{ scaleX: 0 }}
          animate={{ scaleX: 1 }}
          transition={{ duration: 0.5, delay: 0.2, ease: "easeOut" }}
          style={{ transformOrigin: "left" }}
        />
      </motion.div>
    </motion.div>
  );
}

function CompactionDivider() {
  return (
    <div className="flex items-center gap-3 py-2">
      <div className="flex-1 border-t border-purple-400/20" />
      <span className="text-xs font-medium text-purple-400">Context compacted</span>
      <div className="flex-1 border-t border-purple-400/20" />
    </div>
  );
}

function ErrorBar({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
      {message}
    </div>
  );
}

// --- Memoized message list — skips re-render when only input state changes ---

export const MessageList = React.memo(function MessageList({ messages, agentName, agentNamespace, highlightFrom, highlightTo }: { messages: ChatMessage[]; agentName?: string; agentNamespace?: string; highlightFrom?: string; highlightTo?: string }) {
  const userTextCount: Record<string, number> = {};
  const fromTime = highlightFrom ? new Date(highlightFrom).getTime() : null;
  const toTime = highlightTo ? new Date(highlightTo).getTime() : null;
  const [highlightVisible, setHighlightVisible] = useState(!!highlightFrom);
  const fadeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Fade the border 2s after the user scrolls to see the end of the highlighted task.
  const highlightEndRef = useCallback((node: HTMLDivElement | null) => {
    if (!node || fadeTimerRef.current) return;
    const container = node.closest("[data-messages]")?.parentElement;
    if (!container) return;
    const observer = new IntersectionObserver(
      (entries) => {
        // Trigger when the bottom sentinel becomes visible (user reached task end).
        if (entries[0].isIntersecting && !fadeTimerRef.current) {
          fadeTimerRef.current = setTimeout(() => {
            setHighlightVisible(false);
            const url = new URL(window.location.href);
            if (url.searchParams.has("taskFrom")) {
              url.searchParams.delete("taskFrom");
              url.searchParams.delete("taskTo");
              url.searchParams.delete("taskEvents");
              window.history.replaceState(null, "", url.pathname + (url.search || ""));
            }
          }, 2000);
          observer.disconnect();
        }
      },
      { root: container, threshold: 0 }
    );
    observer.observe(node);
  }, []);

  function renderMsg(msg: ChatMessage, i: number) {
    const key = (() => {
      if (msg.kind === "user") {
        const slug = msg.text.slice(0, 80);
        const n = (userTextCount[slug] = (userTextCount[slug] ?? 0) + 1);
        return `user-${slug}-${n}`;
      }
      return `${msg.kind}-${msg.timestamp}-${i}`;
    })();
    switch (msg.kind) {
      case "user":
        return <div key={key}><UserBubble text={msg.text} timestamp={msg.timestamp} /></div>;
      case "text":
        return <AgentBubble key={key} text={msg.text} timestamp={msg.timestamp} usage={msg.usage} agentName={agentName} namespace={agentNamespace} />;
      case "thinking":
        return <ThinkingBubble key={key} text={msg.text} />;
      case "tool":
        return (
          <motion.div key={key} initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.2, ease: "easeOut" }}>
            {msg.toolName === "Skill"
              ? <SkillCard skillName={msg.description ?? "skill"} args={msg.command} />
              : <ToolCard toolName={msg.toolName} description={msg.description} command={msg.command} input={msg.input} output={msg.output} />}
          </motion.div>
        );
      case "completed":
        return <CompletedDivider key={key} costUSD={msg.costUSD} duration={msg.duration} turns={msg.turns} inputTokens={msg.inputTokens} outputTokens={msg.outputTokens} cacheReadTokens={msg.cacheReadTokens} cacheCreationTokens={msg.cacheCreationTokens} />;
      case "cancelled":
        return <CancelledDivider key={key} />;
      case "compaction":
        return <CompactionDivider key={key} />;
      case "error":
        return <ErrorBar key={key} message={msg.message} />;
    }
  }

  // Group messages: highlighted ones go into a single wrapper div.
  if (fromTime == null || toTime == null) {
    return <>{messages.map((msg, i) => renderMsg(msg, i))}</>;
  }

  const elements: React.ReactNode[] = [];
  let highlightBuf: { msg: ChatMessage; idx: number }[] = [];

  const flushHighlight = () => {
    if (highlightBuf.length === 0) return;
    elements.push(
      <div
        key={`hl-${highlightBuf[0].idx}`}
        data-task-highlight=""
        className={`rounded-lg px-3 py-2 flex flex-col gap-3 transition-all duration-500 ${highlightVisible ? "border-2 border-amber-400/30" : "border-2 border-transparent"}`}
      >
        {highlightBuf.map(({ msg, idx }) => renderMsg(msg, idx))}
        <div ref={highlightEndRef} className="h-0" />
      </div>
    );
    highlightBuf = [];
  };

  for (let i = 0; i < messages.length; i++) {
    const msg = messages[i];
    const msgTime = new Date(msg.timestamp).getTime();
    const isHighlighted = msgTime >= fromTime && msgTime <= toTime;

    if (isHighlighted) {
      highlightBuf.push({ msg, idx: i });
    } else {
      flushHighlight();
      elements.push(renderMsg(msg, i));
    }
  }
  flushHighlight();

  return <>{elements}</>;
});

// --- Main component ---

export function AgentChat({
  agentName,
  agentNamespace,
  agentStatus,
  agentLifecycle,
  agentContextWindow,
  events,
  taskStatus,
  initialPending,
  hasMoreEvents,
  loadingOlder,
  onLoadOlder,
  scrollContainerRef: parentScrollRef,
  scrollSnapshotRef,
  highlightTaskFrom,
  highlightTaskTo,
  hasNewerEvents,
  loadingNewer,
  onLoadNewer,
}: AgentChatProps) {
  const [input, setInputRaw] = useState(() => {
    if (typeof window === "undefined") return "";
    return localStorage.getItem(`draft:${agentName}`) ?? "";
  });
  const setInput = useCallback((v: string) => {
    setInputRaw(v);
    if (v) localStorage.setItem(`draft:${agentName}`, v);
    else localStorage.removeItem(`draft:${agentName}`);
  }, [agentName]);
  const [lifecycle, setLifecycleRaw] = useState<"" | "Sleep" | "AutoDelete">((agentLifecycle as "" | "Sleep" | "AutoDelete") || "");
  const setLifecycle = useCallback((lc: "" | "Sleep" | "AutoDelete") => {
    setLifecycleRaw(lc);
    patchAgent(agentName, { lifecycle: lc }, agentNamespace).catch(() => {});
  }, [agentName, agentNamespace]);
  const [lifecycleOpen, setLifecycleOpen] = useState(false);
  const lifecycleRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!lifecycleOpen) return;
    function handleClick(e: MouseEvent) {
      if (lifecycleRef.current && !lifecycleRef.current.contains(e.target as Node)) {
        setLifecycleOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [lifecycleOpen]);
  const [contextWindow, setContextWindow] = useState(agentContextWindow ?? 200000);
  useEffect(() => {
    if (agentContextWindow) setContextWindow(agentContextWindow);
  }, [agentContextWindow]);
  const [pendingText, setPendingText] = useState<string | null>(initialPending ?? null);
  const [pendingTimestamp, setPendingTimestamp] = useState<string>(new Date().toISOString());
  // Persisted user messages that the server doesn't echo back (no task_started event)
  const [localUserMessages, setLocalUserMessages] = useState<ChatMessage[]>(
    initialPending
      ? [{ kind: "user" as const, text: initialPending, timestamp: new Date().toISOString() }]
      : []
  );
  const bottomRef = useRef<HTMLDivElement>(null);
  const bottomSentinelRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const eventCountAtSend = useRef(events.length);
  const forceScrollToBottom = useRef(false);

  // Expose scroll container to parent for scrollHeight snapshots.
  useEffect(() => {
    if (parentScrollRef && scrollContainerRef.current) {
      (parentScrollRef as { current: HTMLElement | null }).current = scrollContainerRef.current;
    }
  });

  const serverMessages = useMemo(() => eventsToChatMessages(events), [events]);

  // Derive working state from events (primary) with polled taskStatus as fallback
  const eventBasedWorking = (() => {
    for (let i = events.length - 1; i >= 0; i--) {
      const t = events[i].type;
      if (t === "task_completed" || t === "task_cancelled" || t === "error") return false;
      if (t === "task_started") return true;
    }
    return taskStatus === "InProgress";
  })();

  // Clear pendingText only when the task finishes — not when the server echoes the message
  // (the memo already dedupes, so clearing on echo would just cause an unnecessary re-render)
  const hasPending = pendingText != null;
  useEffect(() => {
    if (!hasPending) return;
    if (!eventBasedWorking && events.length > eventCountAtSend.current) {
      setPendingText(null);
    }
  }, [hasPending, eventBasedWorking, events.length]);

  const [cancelling, setCancelling] = useState(false);

  // Show thinking: pending send OR actively working
  const isWorking = hasPending || eventBasedWorking;

  // Build messages: merge server messages with local user messages that server didn't echo.
  // Pending message is shown instantly, then replaced by the server echo (same render, no duplicate).
  const messages: ChatMessage[] = useMemo(() => {
    const serverUserTexts = new Set(
      serverMessages.filter((m): m is ChatMessage & { kind: "user"; text: string } => m.kind === "user").map((m) => m.text.trim())
    );
    const missingUserMessages = localUserMessages.filter(
      (m) => m.kind === "user"
        && !serverUserTexts.has(m.text.trim())
        && !(pendingText && m.text.trim() === pendingText.trim() && m.timestamp === pendingTimestamp)
    );

    const all = [...serverMessages, ...missingUserMessages];

    // Add pending message only if server hasn't echoed the same text yet.
    // This ensures instant display on send, and zero-duplicate handoff when server catches up.
    if (pendingText) {
      const echoedByServer = serverUserTexts.has(pendingText.trim());
      if (!echoedByServer) {
        all.push({ kind: "user" as const, text: pendingText, timestamp: pendingTimestamp });
      }
    }

    return all.sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
  }, [serverMessages, localUserMessages, pendingText, pendingTimestamp]);

  const lastInputTokens = (() => {
    for (let i = messages.length - 1; i >= 0; i--) {
      const m = messages[i];
      if (m.kind === "completed" && (m.contextTokens ?? m.inputTokens) != null)
        return m.contextTokens ?? m.inputTokens;
    }
    return undefined;
  })();

  // Restore scroll position when older messages are prepended.
  // Parent snapshots scrollHeight BEFORE state update; useLayoutEffect runs
  // AFTER DOM update but BEFORE paint — so user sees no jump.
  useLayoutEffect(() => {
    const container = scrollContainerRef.current;
    const snapshot = scrollSnapshotRef?.current;
    if (!container || snapshot == null) return;
    container.scrollTop = container.scrollHeight - snapshot;
    (scrollSnapshotRef as { current: number | null }).current = null;
  });

  // Auto-scroll: snap to bottom on initial load, then smooth-scroll only when near bottom
  const initialScrollDone = useRef(false);
  // Reset initial scroll when navigating to a different agent
  useEffect(() => { initialScrollDone.current = false; }, [agentName]);
  const prevMsgCountRef = useRef(0);
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container || messages.length === 0) return;
    const prevCount = prevMsgCountRef.current;
    prevMsgCountRef.current = messages.length;

    if (!initialScrollDone.current) {
      initialScrollDone.current = true;
      if (highlightTaskFrom) {
        // Scroll to the first highlighted message.
        requestAnimationFrame(() => {
          const firstHighlight = container.querySelector("[data-task-highlight]") as HTMLElement | null;
          if (firstHighlight) {
            firstHighlight.scrollIntoView({ behavior: "smooth", block: "start" });
          }
        });
      } else {
        // Normal: snap to bottom
        bottomRef.current?.scrollIntoView();
      }
      return;
    }

    if (messages.length <= prevCount) return;
    // Measure distance BEFORE new content renders — this reflects whether the user was near bottom
    const distFromBottomBefore = container.scrollHeight - container.scrollTop - container.clientHeight;
    if (forceScrollToBottom.current || distFromBottomBefore < 600) {
      forceScrollToBottom.current = false;
      // rAF to scroll after new content is in the DOM
      requestAnimationFrame(() => {
        bottomRef.current?.scrollIntoView({ behavior: "smooth" });
      });
    }
  }, [messages.length]);

  // IntersectionObserver to trigger loading older events when sentinel is visible.
  // Uses refs for all guards so the observer identity is stable (no cascade re-creates).
  const onLoadOlderRef = useRef(onLoadOlder);
  onLoadOlderRef.current = onLoadOlder;
  const hasMoreRef = useRef(hasMoreEvents);
  hasMoreRef.current = hasMoreEvents;
  const loadingOlderRef = useRef(loadingOlder);
  loadingOlderRef.current = loadingOlder;

  // Re-run when messages first load (initialScrollDone becomes true).
  const [observerReady, setObserverReady] = useState(false);
  useEffect(() => {
    if (initialScrollDone.current && !observerReady) setObserverReady(true);
  }, [messages.length, observerReady]);

  useEffect(() => {
    if (!observerReady) return;
    const sentinel = sentinelRef.current;
    const container = scrollContainerRef.current;
    if (!sentinel || !container) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMoreRef.current && !loadingOlderRef.current && onLoadOlderRef.current) {
          onLoadOlderRef.current();
        }
      },
      { root: container, threshold: 0.1 }
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [observerReady, agentName]);

  // Observer for loading newer events (bottom sentinel).
  const onLoadNewerRef = useRef(onLoadNewer);
  onLoadNewerRef.current = onLoadNewer;
  const hasNewerRef = useRef(hasNewerEvents);
  hasNewerRef.current = hasNewerEvents;
  const loadingNewerRef = useRef(loadingNewer);
  loadingNewerRef.current = loadingNewer;

  useEffect(() => {
    if (!observerReady || !hasNewerEvents) return;
    const sentinel = bottomSentinelRef.current;
    const container = scrollContainerRef.current;
    if (!sentinel || !container) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNewerRef.current && !loadingNewerRef.current && onLoadNewerRef.current) {
          onLoadNewerRef.current();
        }
      },
      { root: container, threshold: 0.1 }
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [observerReady, hasNewerEvents, agentName]);

  const handleSend = useCallback(() => {
    const text = input.trim();
    if (!text) return;

    const ts = new Date().toISOString();
    setInput("");
    eventCountAtSend.current = events.length;
    setPendingTimestamp(ts);
    setPendingText(text);
    setLocalUserMessages((prev) => [
      ...prev,
      { kind: "user" as const, text, timestamp: ts },
    ]);
    forceScrollToBottom.current = true;
    // Scroll after React renders the new message
    setTimeout(() => { textareaRef.current?.focus(); bottomRef.current?.scrollIntoView({ behavior: "smooth" }); }, 50);
    // Fire and forget — state is already set, render happens immediately
    createAgent({ name: agentName, instructions: text, namespace: agentNamespace, lifecycle })
      .then((res) => { if (res.modelContextWindow) setContextWindow(res.modelContextWindow); })
      .catch(() => setPendingText(null));
  }, [input, agentName, agentNamespace, lifecycle, events.length]);

  const handleCancel = useCallback(async () => {
    if (!isWorking || cancelling) return;
    setCancelling(true);
    setPendingText(null);
    try {
      await cancelAgent(agentName, agentNamespace);
    } catch {
      setCancelling(false);
    }
  }, [isWorking, cancelling, agentName, agentNamespace]);

  // Reset cancelling when the task actually stops (event arrives)
  useEffect(() => {
    if (cancelling && !eventBasedWorking) {
      setCancelling(false);
    }
  }, [cancelling, eventBasedWorking]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
    if (e.key === "Escape" && isWorking) {
      e.preventDefault();
      handleCancel();
    }
  };

  // Global ESC to cancel
  useEffect(() => {
    if (!isWorking) return;
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        e.preventDefault();
        handleCancel();
      }
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [isWorking, handleCancel]);

  const showThinking = isWorking && !cancelling;

  // Track if user is scrolled away from the bottom
  const [showScrollDown, setShowScrollDown] = useState(false);
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;
    function onScroll() {
      const dist = container!.scrollHeight - container!.scrollTop - container!.clientHeight;
      setShowScrollDown(dist > 300);
    }
    container.addEventListener("scroll", onScroll, { passive: true });
    return () => container.removeEventListener("scroll", onScroll);
  }, []);

  const scrollToBottom = useCallback(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  return (
    <div className="flex h-full flex-1 min-w-0 flex-col">
      {/* Messages area */}
      <div className="relative flex-1 overflow-hidden">
      <div ref={scrollContainerRef} className="h-full overflow-x-hidden overflow-y-auto px-4 pt-4 pb-4">
        {messages.length === 0 && !showThinking ? (
          <div className="flex h-full items-center justify-center text-sm text-[var(--color-text-secondary)]">
            {agentStatus === "Sleeping"
              ? "Agent is sleeping. Send a message to wake it up."
              : "No messages yet. Waiting for events..."}
          </div>
        ) : (
          <div data-messages className="flex flex-col gap-3 min-w-0">
            {/* Sentinel for infinite scroll */}
            <div ref={sentinelRef} className="h-1 shrink-0" />
            {loadingOlder && (
              <div className="flex justify-center py-2">
                <div className="flex items-center gap-2">
                  {[0, 1, 2].map((i) => (
                    <motion.span
                      key={i}
                      className="size-1.5 rounded-full bg-[var(--color-text-secondary)]"
                      animate={{ opacity: [0.3, 1, 0.3] }}
                      transition={{
                        duration: 1,
                        repeat: Infinity,
                        delay: i * 0.15,
                        ease: "easeInOut",
                      }}
                    />
                  ))}
                </div>
              </div>
            )}
            <MessageList messages={messages} agentName={agentName} agentNamespace={agentNamespace} highlightFrom={highlightTaskFrom} highlightTo={highlightTaskTo} />
            <AnimatePresence>
            {showThinking && (
              <motion.div
                className="flex items-center gap-2.5 py-1"
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -12, transition: { duration: 0.25, ease: "easeIn" } }}
                transition={{ duration: 0.2 }}
              >
                <div className="flex items-center gap-1">
                  {[0, 1, 2].map((i) => (
                    <motion.span
                      key={i}
                      className="size-1.5 rounded-full bg-[var(--color-brand-violet)]"
                      animate={{ opacity: [0.3, 1, 0.3], scale: [0.8, 1.1, 0.8] }}
                      transition={{
                        duration: 1.2,
                        repeat: Infinity,
                        delay: i * 0.2,
                        ease: "easeInOut",
                      }}
                    />
                  ))}
                </div>
                <span className="text-xs text-[var(--color-brand-violet-light)]">
                  Thinking
                </span>
              </motion.div>
            )}
            </AnimatePresence>
            {hasNewerEvents && (
              <div ref={bottomSentinelRef} className="h-1 shrink-0" />
            )}
            {loadingNewer && (
              <div className="flex justify-center py-2">
                <span className="text-xs text-[var(--color-text-muted)]">Loading...</span>
              </div>
            )}
            <div ref={bottomRef} />
          </div>
        )}
      </div>
        {/* Scroll to bottom button */}
        <AnimatePresence>
          {showScrollDown && (
            <motion.button
              initial={{ opacity: 0, y: 10, scale: 0.8 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: 10, scale: 0.8 }}
              transition={{ type: "spring", stiffness: 400, damping: 25 }}
              onClick={scrollToBottom}
              className="absolute bottom-4 left-1/2 -translate-x-1/2 z-10 flex size-9 items-center justify-center rounded-full bg-[var(--color-brand-blue)] text-white shadow-[0_4px_16px_rgba(63,133,217,0.4)] hover:bg-[var(--color-brand-blue-light)] hover:shadow-[0_4px_20px_rgba(63,133,217,0.5)] transition-all cursor-pointer"
            >
              <ArrowDown className="size-4" />
            </motion.button>
          )}
        </AnimatePresence>
      </div>

      {/* Input area */}
      <div className="shrink-0 bg-[var(--color-bg)]">
        <ContextBar inputTokens={lastInputTokens} contextWindow={contextWindow} />
        <div className="border-t border-[var(--color-border)]" />
        <div className="flex gap-2 p-4">
          <div className="flex-1 min-w-0">
            <textarea
              ref={textareaRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={cancelling ? "Cancelling..." : isWorking ? "Send a follow-up message..." : "Send a message..."}
              rows={1}
              className="field-sizing-content max-h-24 min-h-10 w-full resize-none break-all overflow-y-auto rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-2.5 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)] focus:border-[var(--color-brand-blue)] focus:outline-none"
            />
          </div>
          <button
            type="button"
            onClick={handleSend}
            disabled={!input.trim()}
            className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-[#3f85d9] text-white transition-opacity hover:opacity-80 disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <ArrowUp className="size-4" />
          </button>
          {(isWorking || cancelling) && (
            <div className="group/stop relative shrink-0">
              <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 opacity-0 group-hover/stop:opacity-100 transition-opacity duration-150 pointer-events-none">
                <div className="whitespace-nowrap rounded-md bg-[var(--color-surface-raised)] border border-[var(--color-border)] px-2 py-1 text-[11px] text-[var(--color-text-secondary)]">
                  Press <kbd className="font-mono font-semibold text-[var(--color-text)]">Esc</kbd> to interrupt
                </div>
              </div>
              <button
                type="button"
                onClick={handleCancel}
                disabled={cancelling}
                className="flex size-10 items-center justify-center rounded-xl bg-red-500 text-white transition-opacity hover:opacity-80 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Square className="size-3.5 fill-current" />
              </button>
            </div>
          )}
          {/* Lifecycle menu */}
          <div className="relative" ref={lifecycleRef}>
            <motion.button
              type="button"
              onClick={() => setLifecycleOpen(!lifecycleOpen)}
              whileTap={{ scale: 0.9 }}
              animate={{
                borderColor: lifecycle === "" ? "var(--color-border)" : lifecycle === "Sleep" ? "rgba(234,179,8,0.4)" : "rgba(239,68,68,0.4)",
                backgroundColor: lifecycle === "" ? "rgba(0,0,0,0)" : lifecycle === "Sleep" ? "rgba(234,179,8,0.1)" : "rgba(239,68,68,0.1)",
                color: lifecycle === "" ? "var(--color-text-secondary)" : lifecycle === "Sleep" ? "#facc15" : "#f87171",
              }}
              transition={{ duration: 0.3 }}
              className="flex size-10 shrink-0 items-center justify-center rounded-xl border hover:opacity-80 cursor-pointer"
              title={`Lifecycle: ${lifecycle || "Default (keep running)"}`}
            >
              <Settings2 className="size-4" />
            </motion.button>
            <AnimatePresence>
              {lifecycleOpen && (
                <motion.div
                  initial={{ opacity: 0, scale: 0.9, y: 4 }}
                  animate={{ opacity: 1, scale: 1, y: 0 }}
                  exit={{ opacity: 0, scale: 0.9, y: 4 }}
                  transition={{ duration: 0.15 }}
                  className="absolute bottom-14 right-0 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-raised)] shadow-[0_8px_32px_rgba(0,0,0,0.4)]"
                >
                  <p className="px-3 pt-2 pb-1 text-[10px] font-semibold uppercase tracking-wider text-[var(--color-text-secondary)] text-center">Lifecycle Mode</p>
                  <div className="flex gap-2 p-2 pt-0">
                  <motion.button
                    type="button"
                    onClick={() => { setLifecycle(""); }}
                    initial={{ opacity: 0, y: 8 }}
                    animate={{
                      opacity: 1, y: 0,
                      backgroundColor: lifecycle === "" ? "rgba(63,133,217,0.15)" : "rgba(0,0,0,0)",
                      color: lifecycle === "" ? "#3f85d9" : "#7c7c98",
                    }}
                    transition={{ backgroundColor: { duration: 0.2 }, color: { duration: 0.2 }, opacity: { duration: 0.15 }, y: { duration: 0.15 } }}
                    className="flex flex-col items-center gap-1 rounded-lg px-3 py-2 text-[10px] font-medium hover:bg-[var(--color-surface-hover)] cursor-pointer"
                    title="Keep running after task"
                  >
                    <Play className="size-4" />
                    Default
                  </motion.button>
                  <motion.button
                    type="button"
                    onClick={() => { setLifecycle("Sleep"); }}
                    initial={{ opacity: 0, y: 8 }}
                    animate={{
                      opacity: 1, y: 0,
                      backgroundColor: lifecycle === "Sleep" ? "rgba(234,179,8,0.15)" : "rgba(0,0,0,0)",
                      color: lifecycle === "Sleep" ? "#facc15" : "#7c7c98",
                    }}
                    transition={{ backgroundColor: { duration: 0.2 }, color: { duration: 0.2 }, opacity: { duration: 0.15, delay: 0.05 }, y: { duration: 0.15, delay: 0.05 } }}
                    className="flex flex-col items-center gap-1 rounded-lg px-3 py-2 text-[10px] font-medium hover:bg-[var(--color-surface-hover)] cursor-pointer"
                    title="Sleep after task (preserve workspace)"
                  >
                    <Moon className="size-4" />
                    Sleep
                  </motion.button>
                  <motion.button
                    type="button"
                    onClick={() => { setLifecycle("AutoDelete"); }}
                    initial={{ opacity: 0, y: 8 }}
                    animate={{
                      opacity: 1, y: 0,
                      backgroundColor: lifecycle === "AutoDelete" ? "rgba(239,68,68,0.15)" : "rgba(0,0,0,0)",
                      color: lifecycle === "AutoDelete" ? "#f87171" : "#7c7c98",
                    }}
                    transition={{ backgroundColor: { duration: 0.2 }, color: { duration: 0.2 }, opacity: { duration: 0.15, delay: 0.1 }, y: { duration: 0.15, delay: 0.1 } }}
                    className="flex flex-col items-center gap-1 rounded-lg px-3 py-2 text-[10px] font-medium hover:bg-[var(--color-surface-hover)] cursor-pointer"
                    title="Delete agent after task"
                  >
                    <Trash2 className="size-4" />
                    AutoDelete
                  </motion.button>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </div>
      </div>
    </div>
  );
}
