"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { AgentEvent } from "@/lib/types";
import { createAgent } from "@/lib/api";
import { CostBadge } from "@/components/shared/cost-badge";
import { CopyButton } from "@/components/shared/copy-button";
import { cn } from "@/lib/utils";
import {
  ChevronRight,
  ArrowUp,
  ArrowDown,
  Terminal,
  FileText,
  Globe,
  Wrench,
  Settings2,
  Moon,
  Trash2,
  Play,
} from "lucide-react";

type AgentChatProps = {
  agentName: string;
  agentNamespace?: string;
  agentStatus: string;
  events: AgentEvent[];
  taskStatus?: string;
  initialPending?: string;
  hasMoreEvents?: boolean;
  loadingOlder?: boolean;
  onLoadOlder?: () => void;
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
  hasMoreEvents,
  loadingOlder,
  onLoadOlder,
}: AgentChatProps) {
  const [input, setInput] = useState("");
  const [lifecycle, setLifecycle] = useState<"" | "Sleep" | "AutoDelete">("");
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
  const [pendingText, setPendingText] = useState<string | null>(initialPending ?? null);
  const [pendingTimestamp, setPendingTimestamp] = useState<string>(new Date().toISOString());
  // Persisted user messages that the server doesn't echo back (no task_started event)
  const [localUserMessages, setLocalUserMessages] = useState<ChatMessage[]>(
    initialPending
      ? [{ kind: "user" as const, text: initialPending, timestamp: new Date().toISOString() }]
      : []
  );
  const bottomRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const eventCountAtSend = useRef(events.length);
  const prevMessagesLenRef = useRef(0);

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
    if (events.length > eventCountAtSend.current) {
      setPendingText(null);
    }
  }, [hasPending, events.length]);

  // Show thinking: pending send OR actively working
  const isWorking = hasPending || eventBasedWorking;

  // Build messages: merge server messages with local user messages that server didn't echo
  const messages: ChatMessage[] = (() => {
    // Start with local user messages that aren't already in server messages
    const serverUserTexts = new Set(
      serverMessages.filter((m) => m.kind === "user").map((m) => m.text.trim())
    );
    const missingUserMessages = localUserMessages.filter(
      (m) => m.kind === "user" && !serverUserTexts.has(m.text.trim())
    );

    // Merge and sort by timestamp
    const all = [...serverMessages, ...missingUserMessages];

    // Add pending optimistic message if not yet echoed
    if (pendingText) {
      const alreadyPresent = all.some(
        (m) => m.kind === "user" && m.text.trim() === pendingText.trim()
      );
      if (!alreadyPresent) {
        all.push({ kind: "user" as const, text: pendingText, timestamp: pendingTimestamp });
      }
    }

    return all.sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
  })();

  // Preserve scroll position when older messages are prepended
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const prevLen = prevMessagesLenRef.current;
    const curLen = messages.length;
    // If messages were added at the top (prepended), restore scroll position
    if (prevLen > 0 && curLen > prevLen) {
      const delta = curLen - prevLen;
      // Check if scroll was near the top (user was scrolling up to load more)
      // We use a threshold to avoid interfering with normal bottom-scroll
      if (container.scrollTop < 200) {
        // Defer to after DOM paint
        requestAnimationFrame(() => {
          // Measure the height of newly prepended items
          const children = container.querySelector("[data-messages]")?.children;
          if (children) {
            let addedHeight = 0;
            for (let i = 0; i < delta && i < children.length; i++) {
              addedHeight += (children[i] as HTMLElement).offsetHeight + 12; // 12 = gap-3
            }
            container.scrollTop = addedHeight;
          }
        });
      }
    }
    prevMessagesLenRef.current = curLen;
  }, [messages.length]);

  // Auto-scroll: snap to bottom on initial load, then smooth-scroll only when near bottom
  const initialScrollDone = useRef(false);
  const prevMsgCountRef = useRef(0);
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container || messages.length === 0) return;
    const prevCount = prevMsgCountRef.current;
    prevMsgCountRef.current = messages.length;

    if (!initialScrollDone.current) {
      // First render with messages — snap to bottom instantly (no smooth)
      bottomRef.current?.scrollIntoView();
      initialScrollDone.current = true;
      return;
    }

    if (messages.length <= prevCount) return;
    // Only auto-scroll if user is near the bottom
    const distFromBottom = container.scrollHeight - container.scrollTop - container.clientHeight;
    if (distFromBottom < 150) {
      bottomRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages.length]);

  // IntersectionObserver to trigger loading older events when sentinel is visible
  // Only attach after the initial scroll to bottom is done, to avoid immediately loading all pages.
  useEffect(() => {
    if (!initialScrollDone.current) return;
    const sentinel = sentinelRef.current;
    const container = scrollContainerRef.current;
    if (!sentinel || !container || !onLoadOlder) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMoreEvents && !loadingOlder) {
          onLoadOlder();
        }
      },
      { root: container, threshold: 0.1 }
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasMoreEvents, loadingOlder, onLoadOlder]);

  const handleSend = useCallback(async () => {
    const text = input.trim();
    if (!text || isWorking) return;

    const ts = new Date().toISOString();
    setInput("");
    eventCountAtSend.current = events.length;
    setPendingTimestamp(ts);
    setPendingText(text);
    setLocalUserMessages((prev) => [
      ...prev,
      { kind: "user" as const, text, timestamp: ts },
    ]);
    try {
      await createAgent({ name: agentName, instructions: text, namespace: agentNamespace, lifecycle });
    } catch {
      setPendingText(null);
    }
  }, [input, isWorking, agentName, agentNamespace, lifecycle, events.length]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const showThinking = isWorking;

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
    <div className="flex h-full flex-col">
      {/* Messages area */}
      <div className="relative flex-1 overflow-hidden">
      <div ref={scrollContainerRef} className="h-full overflow-y-auto px-4 pt-4 pb-4">
        {messages.length === 0 && !showThinking ? (
          <div className="flex h-full items-center justify-center text-sm text-[var(--color-text-secondary)]">
            {agentStatus === "Sleeping"
              ? "Agent is sleeping. Send a message to wake it up."
              : "No messages yet. Waiting for events..."}
          </div>
        ) : (
          <div data-messages className="flex flex-col gap-3">
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
      <div className="shrink-0 border-t border-[var(--color-border)] bg-[var(--color-bg)] p-4">
        <div className="flex items-center gap-2">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Send a message..."
            disabled={isWorking}
            rows={1}
            className="field-sizing-content max-h-24 min-h-10 flex-1 resize-none rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-2.5 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)] focus:border-[var(--color-brand-blue)] focus:outline-none disabled:opacity-50"
          />
          <button
            type="button"
            onClick={handleSend}
            disabled={!input.trim() || isWorking}
            className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-[var(--color-brand-blue)] text-white transition-opacity hover:opacity-80 disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <ArrowUp className="size-4" />
          </button>
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
              className="flex size-9 shrink-0 items-center justify-center rounded-xl border hover:opacity-80"
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
                  className="absolute bottom-12 right-0 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-raised)] shadow-[0_8px_32px_rgba(0,0,0,0.4)]"
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
                    className="flex flex-col items-center gap-1 rounded-lg px-3 py-2 text-[10px] font-medium hover:bg-[var(--color-surface-hover)]"
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
                    className="flex flex-col items-center gap-1 rounded-lg px-3 py-2 text-[10px] font-medium hover:bg-[var(--color-surface-hover)]"
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
                    className="flex flex-col items-center gap-1 rounded-lg px-3 py-2 text-[10px] font-medium hover:bg-[var(--color-surface-hover)]"
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
