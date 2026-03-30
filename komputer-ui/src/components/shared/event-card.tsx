"use client";

import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { AgentEvent } from "@/lib/types";
import { cn } from "@/lib/utils";
import { CostBadge } from "@/components/shared/cost-badge";
import { CopyButton } from "@/components/shared/copy-button";
import { ChevronRight } from "lucide-react";

type EventCardProps = {
  event: AgentEvent;
};

function EventTimestamp({ timestamp }: { timestamp: string }) {
  const time = new Date(timestamp);
  const display = time.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
  return (
    <span className="shrink-0 text-[10px] text-[var(--color-text-secondary)]">
      {display}
    </span>
  );
}

function TextEvent({ event }: EventCardProps) {
  const content = event.payload.content ?? event.payload.text ?? "";
  return (
    <div className="group/msg flex items-start justify-between gap-2">
      <div className="min-w-0 border-l-2 border-[var(--color-brand-blue)] pl-3">
        <div className="prose-chat text-sm text-[var(--color-text)]">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
        </div>
      </div>
      <div className="flex items-center gap-1 shrink-0">
        <div className="opacity-0 group-hover/msg:opacity-100 transition-opacity duration-200">
          <CopyButton text={content} size="md" />
        </div>
        <EventTimestamp timestamp={event.timestamp} />
      </div>
    </div>
  );
}

function ToolResultEvent({ event }: EventCardProps) {
  const [open, setOpen] = useState(false);
  const toolName = event.payload.tool ?? event.payload.name ?? "tool";
  const command = event.payload.input?.command ?? event.payload.input?.cmd;
  const output = event.payload.output ?? event.payload.content ?? "";

  return (
    <div className="overflow-hidden rounded-md border border-[var(--color-border)] bg-[var(--color-surface)] transition-colors duration-150 hover:border-[var(--color-border-hover)] hover:bg-[var(--color-surface-hover)]">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left cursor-pointer"
      >
        <div className="flex items-center gap-2 min-w-0">
          <ChevronRight
            className={cn(
              "size-3.5 shrink-0 text-[var(--color-text-secondary)] transition-transform duration-200",
              open && "rotate-90"
            )}
          />
          <span className="truncate text-sm font-semibold text-[var(--color-text)]">
            {toolName}
          </span>
          {command && (
            <code className="truncate text-xs text-[var(--color-text-secondary)] font-mono">
              {command}
            </code>
          )}
        </div>
        <EventTimestamp timestamp={event.timestamp} />
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
              {event.payload.input && (
                <div className="mb-2">
                  <p className="mb-1 text-[10px] font-medium uppercase text-[var(--color-text-secondary)]">
                    Input
                  </p>
                  <pre className="overflow-x-auto rounded bg-[var(--color-bg)] p-2 font-mono text-xs text-[var(--color-text)]">
                    {typeof event.payload.input === "string"
                      ? event.payload.input
                      : JSON.stringify(event.payload.input, null, 2)}
                  </pre>
                </div>
              )}
              {output && (
                <div>
                  <p className="mb-1 text-[10px] font-medium uppercase text-[var(--color-text-secondary)]">
                    Output
                  </p>
                  <pre className="max-h-60 overflow-auto rounded bg-[var(--color-bg)] p-2 font-mono text-xs text-[var(--color-text)]">
                    {typeof output === "string"
                      ? output
                      : JSON.stringify(output, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

function ThinkingEvent({ event }: EventCardProps) {
  return (
    <div className="flex items-start justify-between gap-2">
      <p className="min-w-0 whitespace-pre-wrap text-xs italic text-[var(--color-brand-violet-light)]/70 border-l-2 border-[var(--color-brand-violet)]/30 pl-2">
        {event.payload.content ?? event.payload.text ?? ""}
      </p>
      <EventTimestamp timestamp={event.timestamp} />
    </div>
  );
}

function InfoBar({
  event,
  color,
  children,
}: EventCardProps & { color: string; children: React.ReactNode }) {
  return (
    <div
      className={cn(
        "flex items-center justify-between gap-2 rounded-md px-3 py-2 text-sm",
        color
      )}
    >
      <div className="flex items-center gap-2 min-w-0">{children}</div>
      <EventTimestamp timestamp={event.timestamp} />
    </div>
  );
}

function TaskStartedEvent({ event }: EventCardProps) {
  return (
    <InfoBar
      event={event}
      color="border border-blue-500/30 bg-blue-500/10 text-blue-400"
    >
      <span>Task started</span>
    </InfoBar>
  );
}

function TaskCompletedEvent({ event }: EventCardProps) {
  return (
    <InfoBar
      event={event}
      color="border border-green-500/30 bg-green-500/10 text-green-400"
    >
      <span>Task completed</span>
      {event.payload.costUSD && <CostBadge cost={event.payload.costUSD} />}
      {event.payload.duration && (
        <span className="text-xs opacity-75">{event.payload.duration}</span>
      )}
      {event.payload.turns != null && (
        <span className="text-xs opacity-75">
          {event.payload.turns} turn{event.payload.turns !== 1 ? "s" : ""}
        </span>
      )}
    </InfoBar>
  );
}

function TaskCancelledEvent({ event }: EventCardProps) {
  return (
    <InfoBar
      event={event}
      color="border border-amber-500/30 bg-amber-500/10 text-amber-400"
    >
      <span>Task cancelled</span>
    </InfoBar>
  );
}

function ErrorEvent({ event }: EventCardProps) {
  return (
    <InfoBar
      event={event}
      color="border border-red-500/30 bg-red-500/10 text-red-400"
    >
      <span>{event.payload.message ?? event.payload.error ?? "Error"}</span>
    </InfoBar>
  );
}

export function EventCard({ event }: EventCardProps) {
  switch (event.type) {
    case "text":
      return <TextEvent event={event} />;
    case "tool_result":
      return <ToolResultEvent event={event} />;
    case "thinking":
      return <ThinkingEvent event={event} />;
    case "task_started":
      return <TaskStartedEvent event={event} />;
    case "task_completed":
      return <TaskCompletedEvent event={event} />;
    case "task_cancelled":
      return <TaskCancelledEvent event={event} />;
    case "error":
      return <ErrorEvent event={event} />;
    case "tool_call":
      return null;
    default:
      return null;
  }
}
