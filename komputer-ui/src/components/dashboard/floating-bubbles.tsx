"use client";

import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { AnimatePresence, motion } from "framer-motion";
import { Brain, Check, MessageSquare, Wrench, X } from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { AgentEvent } from "@/lib/types";
import { useWebSocket } from "@/hooks/use-websocket";

// Buffer between the bubble bottom and the viewport bottom edge — keeps the
// scrollable bubble from bumping right against the screen edge.
const VIEWPORT_BOTTOM_BUFFER = 40;
// Floor on the dynamic max-height so a bubble can never collapse below this
// even if its top is near the viewport bottom (e.g. a tall textarea pushed
// it down). The bubble will overflow the viewport in that edge case, which
// is better than rendering as a useless 1-line scroll surface.
const BUBBLE_MIN_HEIGHT = 120;
// Hard cap on the number of bubbles rendered. Older ones blur-slide out
// via AnimatePresence as new ones arrive. Combined with the per-bubble
// height cap this keeps the column inside the viewport.
const MAX_BUBBLES = 6;
// Reserved space below the last bubble so the Clear button has room to
// render without being clipped or pushing off-screen. Includes the button's
// height + the column's gap-2 (8px).
const CLEAR_BUTTON_RESERVE = 36;

export interface FloatingBubblesProps {
  /** Active streaming session — agent name + namespace + when it started. */
  session: { name: string; namespace: string; startedAt: number } | null;
  /** Called once the websocket has emitted the first content event. The parent
   *  uses this to flip the Go button's spinner back to "Go". */
  onFirstResponse: () => void;
  /** Called when this session's task completes; bubbles persist until next prompt. */
  onTaskComplete: () => void;
  /** Called when the user clicks the Clear bubble — parent drops the session
   *  so the bubbles unmount cleanly. */
  onClear: () => void;
}

interface Bubble {
  id: string;
  kind: "thinking" | "text";
  text: string;
  /** Tool calls fired AFTER this bubble was emitted, before the next bubble. */
  actions: ToolAction[];
}

interface ToolAction {
  id: string;
  tool: string;
  summary: string;
}

/**
 * Renders a floating column of bubbles below the prompt textarea while the
 * agent streams events. The "thinking" bubble is always pinned at the top
 * (or "Done ✓" once the task completes) and message bubbles stack below.
 *
 * Bubble overflow past MAX_BUBBLES blurs out the oldest. Clicking any bubble
 * navigates to the agent's chat page.
 */
export function FloatingBubbles({ session, onFirstResponse, onTaskComplete, onClear }: FloatingBubblesProps) {
  const router = useRouter();
  const { events } = useWebSocket(session?.name ?? null);
  const [bubbles, setBubbles] = useState<Bubble[]>([]);
  const [done, setDone] = useState(false);
  const announcedFirstRef = useRef(false);
  const sessionKeyRef = useRef<string | null>(null);
  // Tracks every event id we've ever consumed for the current session, even
  // after the user clicks Clear. Without this, clearing `bubbles` would let
  // the events effect re-ingest the same events on its next run and the
  // bubbles would pop right back, killing the exit animation.
  const consumedRef = useRef<Set<string>>(new Set());
  const clearedRef = useRef(false);

  // Reset per session.
  const sessionKey = session ? `${session.namespace}/${session.name}/${session.startedAt}` : null;
  useEffect(() => {
    if (sessionKeyRef.current !== sessionKey) {
      sessionKeyRef.current = sessionKey;
      setBubbles([]);
      setDone(false);
      announcedFirstRef.current = false;
      consumedRef.current = new Set();
      clearedRef.current = false;
    }
  }, [sessionKey]);

  // Drain events into bubbles, filtering by session.startedAt so we ignore
  // historical events from past tasks.
  useEffect(() => {
    if (!session) return;
    if (clearedRef.current) return;
    const since = session.startedAt;
    const fresh = events.filter((e) => new Date(e.timestamp).getTime() >= since);
    if (fresh.length === 0) return;

    let nextDone = done;
    let nextBubbles = bubbles;
    let bubblesChanged = false;
    let firstSeen = announcedFirstRef.current;

    for (const ev of fresh) {
      const evId = `${ev.timestamp}:${ev.type}`;
      if (consumedRef.current.has(evId)) continue;
      if (ev.type === "tool_call") {
        const action = eventToAction(ev);
        if (!action) continue;
        consumedRef.current.add(evId);
        if (nextBubbles.length === 0) continue; // no preceding bubble — drop it
        const lastIdx = nextBubbles.length - 1;
        const last = nextBubbles[lastIdx];
        nextBubbles = [
          ...nextBubbles.slice(0, lastIdx),
          { ...last, actions: [...last.actions, action] },
        ];
        bubblesChanged = true;
        continue;
      }
      const next = eventToBubble(ev);
      if (next) {
        consumedRef.current.add(evId);
        nextBubbles = [...nextBubbles, next];
        bubblesChanged = true;
        if (!firstSeen) {
          firstSeen = true;
        }
      }
      if (ev.type === "task_completed" || ev.type === "task_cancelled") {
        consumedRef.current.add(evId);
        nextDone = true;
      }
    }

    if (firstSeen && !announcedFirstRef.current) {
      announcedFirstRef.current = true;
      onFirstResponse();
    }
    if (nextDone && !done) {
      onTaskComplete();
    }
    if (bubblesChanged) setBubbles(nextBubbles);
    if (nextDone !== done) setDone(nextDone);
  }, [events, session, bubbles, done, onFirstResponse, onTaskComplete]);

  function handleClick() {
    if (!session) return;
    router.push(`/agents/${session.name}?namespace=${session.namespace}`);
  }

  // Clearing flow: empty the bubbles + done locally first so AnimatePresence
  // can play the blur-slide-out exit animations on every visible bubble and
  // the status bubble. Once the exit duration passes, hand off to the parent
  // to drop the session entirely (which unmounts this component).
  const [clearing, setClearing] = useState(false);
  function handleClearLocal() {
    if (clearing) return;
    clearedRef.current = true;
    setClearing(true);
    setBubbles([]);
    setDone(false);
    // Match the longest exit transition (450ms) plus a small buffer so the
    // parent doesn't unmount this component before the animations finish.
    const t = setTimeout(() => {
      onClear();
      setClearing(false);
    }, 520);
    return () => clearTimeout(t);
  }

  // Escape hatch: if the agent has been "thinking" without producing any
  // bubbles for more than 10s (e.g. a hung session restored from localStorage
  // after a refresh, or a backend the user can't recover from), reveal the
  // Clear button anyway so the dashboard isn't permanently stuck.
  const [stuck, setStuck] = useState(false);
  useEffect(() => {
    setStuck(false);
    if (!session || bubbles.length > 0 || done || clearing) return;
    const t = setTimeout(() => setStuck(true), 10000);
    return () => clearTimeout(t);
  }, [sessionKey, bubbles.length, done, clearing, session]);

  // Dynamic visible-count clamp. We start at MAX_BUBBLES and let the layout
  // effect below decrement until the column actually fits in the viewport.
  //
  // Important: visibleCount is reset upward only when the *session* changes
  // (a new prompt). Adding a bubble to an existing session does NOT bump it
  // back to MAX_BUBBLES — otherwise the new bubble would render briefly at
  // the bigger count, overflow, then the wrapper would re-evict and the
  // column would visibly jump. Keeping visibleCount steady means a new
  // bubble enters in place of the oldest visible one with a clean cross-
  // fade via AnimatePresence.
  const wrapperRef = useRef<HTMLDivElement>(null);
  const [visibleCount, setVisibleCount] = useState<number>(MAX_BUBBLES);
  useEffect(() => {
    // Session change: reset upward, capped by bubble count + MAX.
    setVisibleCount(Math.min(MAX_BUBBLES, Math.max(1, bubbles.length || MAX_BUBBLES)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionKey]);

  useLayoutEffect(() => {
    function fitToViewport() {
      const el = wrapperRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      const overflow = rect.bottom - (window.innerHeight - VIEWPORT_BOTTOM_BUFFER);
      if (overflow > 0 && visibleCount > 1) {
        // Column too tall — drop the oldest visible bubble. The next render
        // re-runs this effect and continues to shrink until it fits or 1.
        setVisibleCount((c) => Math.max(1, c - 1));
      }
    }
    fitToViewport();
    window.addEventListener("resize", fitToViewport);
    let ro: ResizeObserver | null = null;
    if (typeof ResizeObserver !== "undefined") {
      ro = new ResizeObserver(fitToViewport);
      if (wrapperRef.current) ro.observe(wrapperRef.current);
      ro.observe(document.body);
    }
    return () => {
      window.removeEventListener("resize", fitToViewport);
      ro?.disconnect();
    };
  }, [visibleCount, bubbles.length]);

  if (!session) return null;

  // Precompute the visible slice and its length so we can reference both in
  // the JSX without an IIFE — IIFE-wrapped children sometimes confuse
  // AnimatePresence's child-tracking heuristics in subtle ways.
  const visibleSlice = bubbles.slice(Math.max(0, bubbles.length - visibleCount));
  const visibleSliceLen = visibleSlice.length;

  return (
    // Outer wrapper: pointer-events-none so empty gaps between bubbles don't
    // block clicks on the StatCards behind. Each bubble individually opts back
    // in via pointer-events-auto. The container itself never scrolls — tall
    // bubbles cap their own height with internal scroll instead.
    <div
      ref={wrapperRef}
      className="pointer-events-none absolute left-0 right-0 top-full z-30 mt-12 flex flex-col items-center gap-2"
    >
      {/* Status bubble — thinking, then "Done ✓" with a one-shot glow pump.
          Wrapped in AnimatePresence so it gets a clean blur-slide-out when
          the user clicks Clear (the StatusBubble itself uses initial/animate
          but doesn't define exit; we rely on the wrapper's exit transition
          via a sibling motion.div pattern). */}
      <AnimatePresence initial={false}>
        {!clearing && (
          <motion.div
            key="status"
            initial={{ opacity: 0, y: -8, filter: "blur(4px)" }}
            animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
            exit={{
              opacity: 0,
              transition: { duration: 0.4, ease: "easeOut" },
            }}
            transition={{ duration: 0.32, ease: [0.4, 0, 0.2, 1] }}
          >
            <StatusBubble done={done} onClick={handleClick} />
          </motion.div>
        )}
      </AnimatePresence>

      {/*
        Content bubbles — last MAX_BUBBLES only. Older bubbles blur-slide
        out via AnimatePresence's exit animation as new ones arrive. Each
        rendered bubble's body still caps its height to the remaining
        viewport space, so the column stays inside the screen.
      */}
      {/*
        Default sync mode — exits play in the same flow position as the
        item's last rendered location. Each bubble's own exit transition
        independently drives opacity/transform/filter; the layout flow
        collapses only after the last exit finishes.
      */}
      <AnimatePresence initial={false} mode="popLayout">
        {visibleSlice.map((b, i) => (
          <motion.div
            key={b.id}
            initial={{ opacity: 0, y: -12, scale: 0.96, filter: "blur(6px)" }}
            animate={{ opacity: 1, y: 0, scale: 1, filter: "blur(0px)" }}
            exit={{
              opacity: 0,
              y: -12,
              scale: 0.96,
              filter: "blur(6px)",
              transition: { duration: 0.45, ease: "easeOut" },
            }}
            transition={{ duration: 0.36, ease: [0.4, 0, 0.2, 1] }}
            onClick={handleClick}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                handleClick();
              }
            }}
            className="pointer-events-auto group relative flex w-fit max-w-[min(100%,40rem)] cursor-pointer items-start gap-2 rounded-2xl border border-[var(--color-border)]/60 bg-[var(--color-surface)]/40 px-3.5 py-2 text-left text-sm text-[var(--color-text)] shadow-[0_4px_24px_rgba(var(--shadow-color),var(--shadow-strength))] backdrop-blur-md transition-colors hover:border-[var(--color-brand-blue)]/50 hover:bg-[var(--color-surface)]/60"
          >
            <BubbleIcon kind={b.kind} />
            <BubbleBody
              text={b.text}
              bubbleCount={bubbles.length}
              isLast={i === visibleSliceLen - 1}
            />
            <ToolActionsChip actions={b.actions} />
          </motion.div>
        ))}
        {(bubbles.length > 0 || stuck) && (
          <motion.button
            type="button"
            key="__clear__"
            initial={{ opacity: 0, y: -8, filter: "blur(4px)" }}
            animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
            exit={{
              opacity: 0,
              transition: { duration: 0.4, ease: "easeOut" },
            }}
            transition={{ duration: 0.24, ease: "easeOut" }}
            onClick={(e) => {
              e.stopPropagation();
              handleClearLocal();
            }}
            className="pointer-events-auto inline-flex cursor-pointer items-center gap-1.5 rounded-full border border-[var(--color-border)]/60 bg-[var(--color-surface)]/40 px-2.5 py-0.5 text-[11px] text-[var(--color-text-secondary)] shadow-[0_4px_16px_rgba(var(--shadow-color),var(--shadow-strength))] backdrop-blur-md transition-colors hover:border-red-400/50 hover:text-red-300"
          >
            <X className="size-3" />
            Clear
          </motion.button>
        )}
      </AnimatePresence>
    </div>
  );
}

/**
 * Tiny circle pinned to the bottom-right corner of a bubble showing how
 * many tool calls fired after that bubble was emitted. Hovering reveals
 * a scrollable tooltip listing each action (tool name + brief input).
 */
function ToolActionsChip({ actions }: { actions: ToolAction[] }) {
  const [open, setOpen] = useState(false);
  if (actions.length === 0) return null;
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.6, filter: "blur(3px)" }}
      animate={{ opacity: 1, scale: 1, filter: "blur(0px)" }}
      transition={{ duration: 0.24, ease: [0.4, 0, 0.2, 1] }}
      className="pointer-events-auto absolute -bottom-1.5 -right-1.5 z-10"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={() => setOpen(false)}
      onClick={(e) => e.stopPropagation()}
    >
      <button
        type="button"
        className="relative flex size-5 items-center justify-center overflow-hidden rounded-full border border-[var(--color-border-hover)] bg-[var(--color-surface-raised)] text-[var(--color-text)] shadow-[0_2px_8px_rgba(var(--shadow-color),var(--shadow-strength))] transition-colors hover:border-[var(--color-text)]"
        aria-label={`${actions.length} action${actions.length === 1 ? "" : "s"}`}
      >
        <AnimatePresence mode="popLayout" initial={false}>
          <motion.span
            key={actions.length}
            initial={{ y: 6, opacity: 0, filter: "blur(3px)" }}
            animate={{ y: 0, opacity: 1, filter: "blur(0px)" }}
            exit={{ y: -6, opacity: 0, filter: "blur(3px)" }}
            transition={{ duration: 0.22, ease: [0.4, 0, 0.2, 1] }}
            style={{ fontVariantNumeric: "tabular-nums" }}
            className="text-[10px] font-semibold leading-none text-[var(--color-text)]"
          >
            {actions.length}
          </motion.span>
        </AnimatePresence>
      </button>
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, x: -4, filter: "blur(4px)" }}
            animate={{ opacity: 1, x: 0, filter: "blur(0px)" }}
            exit={{ opacity: 0, x: -4, filter: "blur(4px)" }}
            transition={{ duration: 0.18, ease: "easeOut" }}
            className="absolute bottom-1/2 left-1/2 ml-2 mb-2 w-72 rounded-xl border border-[var(--color-border)]/60 bg-[var(--color-surface-raised)] p-2 shadow-[0_8px_32px_rgba(var(--shadow-color),var(--shadow-strength))]"
            role="tooltip"
          >
            <div className="px-2 pb-1.5 pt-0.5 text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">
              Actions ({actions.length})
            </div>
            <div className="max-h-64 overflow-y-auto pr-1" style={{ overscrollBehavior: "contain" }}>
              <ul className="flex flex-col gap-1">
                {actions.map((a) => (
                  <li
                    key={a.id}
                    className="flex items-start gap-2 rounded-md px-2 py-1.5 text-xs hover:bg-[var(--color-surface)]/60"
                  >
                    <Wrench className="mt-0.5 size-3 shrink-0 text-[var(--color-brand-violet)]" />
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-medium text-[var(--color-text)]">{a.tool}</div>
                      {a.summary && (
                        <div className="truncate text-[11px] text-[var(--color-text-muted)]">
                          {a.summary}
                        </div>
                      )}
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}

/**
 * Markdown body for a bubble. Measures its own viewport position and caps
 * its max-height so the bubble never extends past `viewport bottom -
 * VIEWPORT_BOTTOM_BUFFER`. Recomputes on resize, on its own size changes,
 * and whenever the total bubble count changes (since other bubbles above
 * shift this one's top edge).
 */
function BubbleBody({
  text,
  bubbleCount,
  isLast,
}: {
  text: string;
  bubbleCount: number;
  isLast: boolean;
}) {
  const ref = useRef<HTMLDivElement>(null);

  // Apply max-height synchronously via the DOM (not React state) so the
  // parent's eviction effect sees the final heights when it measures the
  // wrapper. Going through state would schedule a second render and the
  // parent's first measurement would see oversized bubbles, leading to
  // over-eviction of older ones.
  //
  // When this is the last visible bubble, reserve extra room below for the
  // Clear button so the button doesn't get pushed off-screen.
  //
  // When bubbleCount === 0 the parent has cleared and all currently mounted
  // bubbles are exiting via AnimatePresence. We must NOT touch DOM styles
  // during exit — synchronous style writes here force layout/paint cycles
  // that interrupt framer-motion's exit interpolation, which is why the
  // exit looked abrupt instead of fading.
  useLayoutEffect(() => {
    if (bubbleCount === 0) return;
    function measure() {
      const el = ref.current;
      if (!el) return;
      const top = el.getBoundingClientRect().top;
      const reserve = VIEWPORT_BOTTOM_BUFFER + (isLast ? CLEAR_BUTTON_RESERVE : 0);
      const available = window.innerHeight - top - reserve;
      el.style.maxHeight = `${Math.max(BUBBLE_MIN_HEIGHT, available)}px`;
    }
    measure();
    window.addEventListener("resize", measure);
    let ro: ResizeObserver | null = null;
    if (typeof ResizeObserver !== "undefined" && ref.current) {
      ro = new ResizeObserver(measure);
      ro.observe(document.body);
    }
    return () => {
      window.removeEventListener("resize", measure);
      ro?.disconnect();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [bubbleCount, isLast]);

  return (
    <div
      ref={ref}
      className="prose-chat min-w-0 flex-1 overflow-y-auto pr-1 text-sm leading-relaxed"
      style={{ maxHeight: BUBBLE_MIN_HEIGHT, overscrollBehavior: "contain" }}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{text}</ReactMarkdown>
    </div>
  );
}

function StatusBubble({ done, onClick }: { done: boolean; onClick: () => void }) {
  // Fire the one-shot glow pump animation only when `done` transitions from
  // false -> true. We track that via a ref so re-renders after the transition
  // don't replay the animation.
  const wasDoneRef = useRef(done);
  const [pumping, setPumping] = useState(false);
  useEffect(() => {
    if (done && !wasDoneRef.current) {
      setPumping(true);
      const t = setTimeout(() => setPumping(false), 1900);
      return () => clearTimeout(t);
    }
    wasDoneRef.current = done;
  }, [done]);

  return (
    <motion.button
      type="button"
      layout
      initial={{ opacity: 0, y: -8, filter: "blur(4px)" }}
      animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
      transition={{ duration: 0.28, ease: "easeOut" }}
      onClick={onClick}
      className={`pointer-events-auto inline-flex cursor-pointer items-center gap-2 rounded-full border bg-[var(--color-surface-raised)] px-3 py-1 text-xs text-[var(--color-text-secondary)] shadow-[0_4px_16px_rgba(var(--shadow-color),var(--shadow-strength))] transition-colors hover:text-[var(--color-text)] ${
        done
          ? "border-emerald-400/60 hover:border-emerald-300"
          : "border-[var(--color-border)] hover:border-[var(--color-brand-blue)]/50"
      } ${pumping ? "animate-done-glow-pump" : ""}`}
    >
      <AnimatePresence mode="wait" initial={false}>
        {done ? (
          <motion.span
            key="done"
            initial={{ opacity: 0, scale: 0.6 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.6 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
            className="flex items-center gap-1.5"
          >
            <Check className="size-3.5 text-emerald-400 animate-check-pop" />
            <span className="text-[var(--color-text)]">Done</span>
          </motion.span>
        ) : (
          <motion.span
            key="thinking"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.18 }}
            className="flex items-center gap-1.5"
          >
            <Brain className="size-3.5 text-[var(--color-brand-blue)]" />
            <span>Thinking</span>
            <span className="inline-flex gap-0.5">
              <span className="thinking-dot thinking-dot-1 size-1 rounded-full bg-[var(--color-text-secondary)]" />
              <span className="thinking-dot thinking-dot-2 size-1 rounded-full bg-[var(--color-text-secondary)]" />
              <span className="thinking-dot thinking-dot-3 size-1 rounded-full bg-[var(--color-text-secondary)]" />
            </span>
          </motion.span>
        )}
      </AnimatePresence>
    </motion.button>
  );
}

function BubbleIcon({ kind }: { kind: Bubble["kind"] }) {
  if (kind === "thinking") {
    return <Brain className="mt-0.5 size-3.5 shrink-0 text-[var(--color-brand-violet)]" />;
  }
  return <MessageSquare className="mt-0.5 size-3.5 shrink-0 text-[var(--color-brand-blue)]" />;
}

function eventToBubble(ev: AgentEvent): Bubble | null {
  const id = `${ev.timestamp}:${ev.type}`;
  if (ev.type === "thinking") {
    const text = textFromPayload(ev.payload);
    if (!text) return null;
    return { id, kind: "thinking", text, actions: [] };
  }
  if (ev.type === "text") {
    const text = textFromPayload(ev.payload);
    if (!text) return null;
    return { id, kind: "text", text, actions: [] };
  }
  return null;
}

function eventToAction(ev: AgentEvent): ToolAction | null {
  const id = `${ev.timestamp}:${ev.type}`;
  const rawTool = (ev.payload?.tool ?? ev.payload?.name) as string | undefined;
  if (!rawTool) return null;
  const tool = prettyToolName(rawTool);
  const summary = summarizeInput(ev.payload?.input);
  return { id, tool, summary };
}

function prettyToolName(tool: string): string {
  // Strip MCP namespace prefixes like "mcp__komputer__create_agent" → "create_agent".
  const parts = tool.split("__");
  return parts[parts.length - 1] || tool;
}

function summarizeInput(input: unknown): string {
  if (input == null) return "";
  if (typeof input === "string") return truncate(input, 80);
  if (typeof input !== "object") return String(input);
  const obj = input as Record<string, unknown>;
  const preferred = ["command", "cmd", "file_path", "path", "query", "url", "name", "prompt"];
  for (const key of preferred) {
    const v = obj[key];
    if (typeof v === "string" && v.trim()) return truncate(v, 80);
  }
  const firstString = Object.values(obj).find((v) => typeof v === "string" && (v as string).trim());
  if (typeof firstString === "string") return truncate(firstString, 80);
  try {
    return truncate(JSON.stringify(obj), 80);
  } catch {
    return "";
  }
}

function truncate(s: string, n: number): string {
  const trimmed = s.trim().replace(/\s+/g, " ");
  return trimmed.length > n ? trimmed.slice(0, n - 1) + "…" : trimmed;
}

function textFromPayload(payload: Record<string, unknown>): string {
  const v = payload?.content ?? payload?.text;
  if (typeof v !== "string") return "";
  return v.trim();
}
