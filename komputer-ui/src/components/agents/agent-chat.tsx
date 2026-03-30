"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { AgentEvent } from "@/lib/types";
import { createAgent } from "@/lib/api";
import { CostBadge } from "@/components/shared/cost-badge";
import { CopyButton } from "@/components/shared/copy-button";
import { Button } from "@/components/kit/button";
import { cn } from "@/lib/utils";
import {
  ChevronRight,
  Send,
  Terminal,
  FileText,
  Globe,
  Wrench,
} from "lucide-react";

type AgentChatProps = {
  agentName: string;
  agentNamespace?: string;
  agentStatus: string;
  events: AgentEvent[];
  taskStatus?: string;
  initialPending?: string;
};

// --- Message types derived from events ---

type ChatMessage =
  | { kind: "user"; text: string; timestamp: string }
  | { kind: "text"; text: string; timestamp: string }
  | { kind: "thinking"; text: string; timestamp: string }
  | {
      kind: "tool";
      toolName: string;
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
      timestamp: string;
    }
  | { kind: "error"; message: string; timestamp: string }
  | { kind: "cancelled"; timestamp: string };

function eventsToChatMessages(events: AgentEvent[]): ChatMessage[] {
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
      case "text": {
        const content = event.payload.content ?? event.payload.text ?? "";
        if (content) {
          messages.push({ kind: "text", text: content, timestamp: event.timestamp });
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
          });
        }
        break;
      }
      case "tool_result": {
        const toolName = event.payload.tool ?? event.payload.name ?? "tool";
        const command =
          event.payload.input?.command ?? event.payload.input?.cmd;
        messages.push({
          kind: "tool",
          toolName,
          command,
          input: event.payload.input,
          output: event.payload.output ?? event.payload.content,
          timestamp: event.timestamp,
        });
        break;
      }
      case "task_completed": {
        const costRaw = event.payload.costUSD ?? event.payload.cost_usd;
        const durationMs = event.payload.duration ?? event.payload.duration_ms;
        const duration = typeof durationMs === "number" ? `${(durationMs / 1000).toFixed(1)}s` : durationMs;
        const cost = typeof costRaw === "number" ? costRaw.toFixed(4) : costRaw;
        messages.push({
          kind: "completed",
          costUSD: cost,
          duration,
          turns: event.payload.turns ?? event.payload.num_turns,
          timestamp: event.timestamp,
        });
        break;
      }
      case "task_cancelled":
        messages.push({ kind: "cancelled", timestamp: event.timestamp });
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

function getToolIcon(name: string) {
  const lower = name.toLowerCase();
  if (lower.includes("bash") || lower.includes("shell"))
    return <Terminal className="size-3.5" />;
  if (lower.includes("read") || lower.includes("write") || lower.includes("edit"))
    return <FileText className="size-3.5" />;
  if (lower.includes("web") || lower.includes("fetch") || lower.includes("curl"))
    return <Globe className="size-3.5" />;
  return <Wrench className="size-3.5" />;
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
        <div className="prose-chat text-sm text-[var(--color-text)] break-all">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{text}</ReactMarkdown>
        </div>
        <p className="mt-1 text-right text-[10px] text-[var(--color-text-secondary)]">
          {new Date(timestamp).toLocaleTimeString([], {
            hour: "2-digit",
            minute: "2-digit",
          })}
        </p>
      </div>
    </motion.div>
  );
}

function AgentBubble({ text, timestamp }: { text: string; timestamp: string }) {
  return (
    <motion.div
      className="group/msg flex justify-start"
      initial={{ opacity: 0, x: -12 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.25, ease: "easeOut" }}
    >
      <div className="relative max-w-[80%] rounded-xl rounded-bl-sm border-l-2 border-[var(--color-brand-blue)] bg-[var(--color-surface)] px-4 py-2.5 transition-[background-color,border-color] duration-150 hover:bg-[var(--color-surface-hover)] hover:border-[var(--color-brand-blue-light)]">
        <div className="absolute -right-8 top-1 opacity-0 group-hover/msg:opacity-100 transition-opacity duration-200">
          <CopyButton text={text} />
        </div>
        <div className="prose-chat text-sm text-[var(--color-text)]">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{text}</ReactMarkdown>
        </div>
        <p className="mt-1 text-[10px] text-[var(--color-text-secondary)]">
          {new Date(timestamp).toLocaleTimeString([], {
            hour: "2-digit",
            minute: "2-digit",
          })}
        </p>
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

function ToolCard({
  toolName,
  command,
  input,
  output,
}: {
  toolName: string;
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
        <span className="truncate text-sm font-semibold text-[var(--color-text)]">
          {toolName}
        </span>
        {command && (
          <code className="truncate text-xs font-mono text-[var(--color-text-secondary)]">
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
              {input != null && (
                <div className="mb-2">
                  <p className="mb-1 text-[10px] font-medium uppercase text-[var(--color-text-secondary)]">
                    Input
                  </p>
                  <pre className="overflow-x-auto rounded bg-[var(--color-bg)] p-2 font-mono text-xs text-[var(--color-text)]">
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

function CompletedDivider({
  costUSD,
  duration,
  turns,
}: {
  costUSD?: string;
  duration?: string;
  turns?: number;
}) {
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
        </motion.div>
      )}
    </motion.div>
  );
}

function CancelledDivider() {
  return (
    <div className="flex items-center gap-3 py-1">
      <div className="flex-1 border-t border-[var(--color-border)]" />
      <span className="text-xs text-amber-400">Task cancelled</span>
      <div className="flex-1 border-t border-[var(--color-border)]" />
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

// --- Main component ---

export function AgentChat({
  agentName,
  agentNamespace,
  agentStatus,
  events,
  taskStatus,
  initialPending,
}: AgentChatProps) {
  const [input, setInput] = useState("");
  const [pendingText, setPendingText] = useState<string | null>(initialPending ?? null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const eventCountAtSend = useRef(events.length);

  const serverMessages = eventsToChatMessages(events);

  // Derive working state from events (primary) with polled taskStatus as fallback
  const eventBasedWorking = (() => {
    for (let i = events.length - 1; i >= 0; i--) {
      const t = events[i].type;
      if (t === "task_completed" || t === "task_cancelled" || t === "error") return false;
      if (t === "task_started") return true;
    }
    return taskStatus === "InProgress";
  })();

  // pendingText is set on send, cleared when new events arrive for this task
  const hasPending = pendingText != null;
  useEffect(() => {
    if (!hasPending) return;
    // Only clear once new events have arrived since the send
    if (events.length > eventCountAtSend.current && eventBasedWorking) {
      setPendingText(null);
    }
    // Also clear if task completed after our send
    if (events.length > eventCountAtSend.current && !eventBasedWorking) {
      setPendingText(null);
    }
  }, [hasPending, events.length, eventBasedWorking]);

  // Show thinking: pending send OR actively working
  const isWorking = hasPending || eventBasedWorking;

  // Build messages: append optimistic user message if pending and not yet in server messages
  const messages: ChatMessage[] = (() => {
    if (!pendingText) return serverMessages;
    return [
      ...serverMessages,
      { kind: "user" as const, text: pendingText, timestamp: new Date().toISOString() },
    ];
  })();

  // Auto-scroll on new messages
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages.length]);

  const handleSend = useCallback(async () => {
    const text = input.trim();
    if (!text || isWorking) return;

    setInput("");
    eventCountAtSend.current = events.length;
    setPendingText(text);
    try {
      await createAgent({ name: agentName, instructions: text, namespace: agentNamespace });
    } catch {
      setPendingText(null);
    }
  }, [input, isWorking, agentName, agentNamespace, events.length]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const showThinking = isWorking;

  return (
    <div className="flex h-full flex-col">
      {/* Messages area */}
      <div className="flex-1 overflow-y-auto px-4 pt-4 pb-4">
        {messages.length === 0 && !showThinking ? (
          <div className="flex h-full items-center justify-center text-sm text-[var(--color-text-secondary)]">
            {agentStatus === "Sleeping"
              ? "Agent is sleeping. Send a message to wake it up."
              : "No messages yet. Waiting for events..."}
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            <AnimatePresence initial={false}>
            {messages.map((msg, i) => {
              switch (msg.kind) {
                case "user":
                  return (
                    <UserBubble
                      key={`${msg.timestamp}-${i}`}
                      text={msg.text}
                      timestamp={msg.timestamp}
                    />
                  );
                case "text":
                  return (
                    <AgentBubble
                      key={`${msg.timestamp}-${i}`}
                      text={msg.text}
                      timestamp={msg.timestamp}
                    />
                  );
                case "thinking":
                  return (
                    <ThinkingBubble
                      key={`${msg.timestamp}-${i}`}
                      text={msg.text}
                    />
                  );
                case "tool":
                  return (
                    <ToolCard
                      key={`${msg.timestamp}-${i}`}
                      toolName={msg.toolName}
                      command={msg.command}
                      input={msg.input}
                      output={msg.output}
                    />
                  );
                case "completed":
                  return (
                    <CompletedDivider
                      key={`${msg.timestamp}-${i}`}
                      costUSD={msg.costUSD}
                      duration={msg.duration}
                      turns={msg.turns}
                    />
                  );
                case "cancelled":
                  return <CancelledDivider key={`${msg.timestamp}-${i}`} />;
                case "error":
                  return (
                    <ErrorBar
                      key={`${msg.timestamp}-${i}`}
                      message={msg.message}
                    />
                  );
              }
            })}
            </AnimatePresence>
            {showThinking && (
              <motion.div
                className="flex items-center gap-2.5 py-1"
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
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
            <div ref={bottomRef} />
          </div>
        )}
      </div>

      {/* Input area */}
      <div className="shrink-0 border-t border-[var(--color-border)] bg-[var(--color-bg)] p-4">
        <div className="flex items-end gap-2">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Send a message..."
            disabled={isWorking}
            rows={1}
            className="field-sizing-content max-h-24 min-h-9 flex-1 resize-none rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)] focus:border-[var(--color-brand-blue)] focus:outline-none disabled:opacity-50"
          />
          <Button
            size="icon"
            onClick={handleSend}
            disabled={!input.trim() || isWorking}
            className="shrink-0 bg-[var(--color-brand-blue)] text-white hover:bg-[var(--color-brand-blue)]/80"
          >
            <Send className="size-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
