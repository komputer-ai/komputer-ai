"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Loader2, Send, FolderOpen } from "lucide-react";
import type { AgentResponse } from "@/lib/types";
import { listAgents, createAgent, patchAgent } from "@/lib/api";
import { Button } from "@/components/kit/button";
import { useTypewriterPlaceholder } from "@/hooks/use-typewriter-placeholder";
import { ActiveAgentChip } from "./active-agent-chip";
import { NamespaceChip } from "./namespace-chip";
import { ModelChip } from "./model-chip";
import { FloatingBubbles } from "./floating-bubbles";

const PERSONAL_AGENT_LABEL = "komputer.ai/personal-agent";
const SESSION_STORAGE_KEY = "personal-agent-session";
const SESSION_TTL_MS = 30 * 60 * 1000; // 30 minutes — enough to bridge a refresh, not stale forever

interface StreamingSession {
  name: string;
  namespace: string;
  startedAt: number;
  expiresAt: number;
}

function loadSession(): StreamingSession | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(SESSION_STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as StreamingSession;
    if (typeof parsed?.expiresAt !== "number" || parsed.expiresAt < Date.now()) {
      localStorage.removeItem(SESSION_STORAGE_KEY);
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

function saveSession(s: StreamingSession): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(s));
  } catch {
    // localStorage unavailable; ignore.
  }
}

function clearSession(): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.removeItem(SESSION_STORAGE_KEY);
  } catch {
    // ignore
  }
}
const EXAMPLE_PROMPTS = [
  "Spin up 3 agents to investigate, fix, and verify why staging is failing — share a workspace and report back when green.",
  "Every weekday at 9am, scan my Linear inbox and post the top 3 priorities to #standup.",
  "Create a 'security-audit' skill from the OWASP top-10 and attach it to every agent labeled team=core.",
  "Sleep every agent idle for over 4 hours and tell me which ones cost the most this week.",
  "Find today's most expensive agent, look at what it ran, and decide whether to keep it alive or sleep it.",
];
const PILL_PROMPTS = [
  { label: "Staging investigation", prompt: EXAMPLE_PROMPTS[0] },
  { label: "Daily standup", prompt: EXAMPLE_PROMPTS[1] },
  { label: "Roll out skill", prompt: EXAMPLE_PROMPTS[2] },
  { label: "Cleanup idle", prompt: EXAMPLE_PROMPTS[3] },
  { label: "Cost watch", prompt: EXAMPLE_PROMPTS[4] },
];

export interface PersonalAgentPromptProps {
  /** Notifies the parent whenever a streaming session starts or ends. The
   *  page uses this to subtly blur the rest of the dashboard while bubbles
   *  are visible, drawing focus to the prompt + bubble column. */
  onSessionActiveChange?: (active: boolean) => void;
}

export function PersonalAgentPrompt({ onSessionActiveChange }: PersonalAgentPromptProps = {}) {
  const [agents, setAgents] = useState<AgentResponse[]>([]);
  const [agentsLoaded, setAgentsLoaded] = useState(false);
  const [active, setActive] = useState<AgentResponse | null>(null);
  // True when the user has chosen "✨ New personal agent" but hasn't clicked Go yet.
  const [isNew, setIsNew] = useState(false);
  // Namespace for the *new* agent (only relevant when isNew is true). Defaults to "default".
  const [newNamespace, setNewNamespace] = useState("default");
  // Model for the *new* agent (only relevant when isNew is true).
  const [newModel, setNewModel] = useState("claude-sonnet-4-6");
  const [prompt, setPrompt] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  // submitting: Go was clicked, request in flight or first event not yet received.
  const [submitting, setSubmitting] = useState(false);
  // Bumping this value re-keys the textarea so a fresh animation plays.
  // The `key` change forces React to unmount/remount the textarea, which
  // triggers the CSS animation cleanly without React reconciliation
  // stripping the animation class from a still-mounted element.
  const [blurInToken, setBlurInToken] = useState(0);
  const [error, setError] = useState<string | null>(null);
  // Active streaming session — when non-null, the bubbles render below the textarea.
  const [session, setSession] = useState<StreamingSession | null>(null);

  const placeholder = useTypewriterPlaceholder({
    prompts: EXAMPLE_PROMPTS,
    // Pause while the user is typing OR while a bubble session is active —
    // we don't want the typewriter ghosting back into the empty textarea
    // while the agent is still streaming or the bubbles are still on screen.
    paused: prompt.length > 0 || session !== null,
  });

  // Auto-grow the textarea to fit content, capped by CSS max-height.
  // Suspends the height transition during the measure phase so the visible
  // animation is current -> target only. Also toggles vertical overflow:
  // hidden while the content fits, auto once we hit the max-height cap.
  // Without this, a 1px scrollHeight rounding quirk shows a phantom scrollbar.
  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    const prev = el.style.transition;
    el.style.transition = "none";
    el.style.height = "auto";
    el.style.overflowY = "hidden"; // measure without a phantom scrollbar
    const contentH = el.scrollHeight;
    const maxH = parseFloat(getComputedStyle(el).maxHeight);
    void el.offsetHeight;
    el.style.transition = prev;
    if (Number.isFinite(maxH) && contentH > maxH) {
      el.style.height = `${maxH}px`;
      el.style.overflowY = "auto";
    } else {
      el.style.height = `${contentH}px`;
      // overflowY stays hidden — content fits.
    }
  }, [prompt]);

  // Sort agents by completionTime desc, createdAt desc fallback.
  const sorted = useMemo(() => {
    const ts = (a: AgentResponse) => {
      const c = a.completionTime ? new Date(a.completionTime).getTime() : 0;
      const cr = a.createdAt ? new Date(a.createdAt).getTime() : 0;
      return c || cr;
    };
    return [...agents].sort((a, b) => ts(b) - ts(a));
  }, [agents]);

  // Pick the most recent personal agent once the list has loaded. Gated on
  // agentsLoaded so we don't prematurely fall into "new" mode on the empty
  // initial-render state before the fetch resolves.
  useEffect(() => {
    if (!agentsLoaded || active || isNew) return;
    if (sorted.length > 0) {
      setActive(sorted[0]);
    } else {
      // No personal agents exist — start in "new" mode so Go creates the first one.
      setIsNew(true);
    }
  }, [agentsLoaded, sorted, active, isNew]);

  // Initial load.
  useEffect(() => {
    listAgents(undefined, { labelSelectors: [`${PERSONAL_AGENT_LABEL}=true`] })
      .then((res) => setAgents(res.agents ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : "failed to load personal agents"))
      .finally(() => setAgentsLoaded(true));
  }, []);

  function nextManagerName(existing: AgentResponse[]): string {
    let max = 0;
    for (const a of existing) {
      const m = a.name.match(/^platform-manager-(\d+)$/);
      if (m) {
        const n = parseInt(m[1], 10);
        if (n > max) max = n;
      }
    }
    return `platform-manager-${max + 1}`;
  }

  function handleSelectExisting(a: AgentResponse) {
    setActive(a);
    setIsNew(false);
  }

  function handleSelectNew() {
    setIsNew(true);
    setActive(null);
  }

  // Update the chip-selected model. When no agent is selected (new-agent
  // mode) just stash it for the upcoming create. For an existing agent,
  // optimistically update the local state, persist via patchAgent, and
  // roll back on failure.
  function handleModelChange(model: string) {
    if (isNew || !active) {
      setNewModel(model);
      return;
    }
    const previous = active.model;
    if (previous === model) return;
    const updated = { ...active, model };
    setActive(updated);
    setAgents((prev) =>
      prev.map((a) =>
        a.name === active.name && a.namespace === active.namespace ? updated : a,
      ),
    );
    patchAgent(active.name, { model }, active.namespace).catch((e) => {
      setError(e instanceof Error ? e.message : "failed to update model");
      // Roll back on failure.
      const rolledBack = { ...active, model: previous };
      setActive(rolledBack);
      setAgents((prev) =>
        prev.map((a) =>
          a.name === active.name && a.namespace === active.namespace ? rolledBack : a,
        ),
      );
    });
  }

  // Animate the prompt text blurring out, then clear the value. Returns a
  // promise that resolves when the visual transition is done so the caller
  // can sequence subsequent UI updates. If the textarea isn't mounted (rare),
  // clears synchronously.
  const blurOutPrompt = useCallback(() => {
    return new Promise<void>((resolve) => {
      const el = textareaRef.current;
      if (!el) {
        setPrompt("");
        resolve();
        return;
      }
      el.classList.remove("animate-text-blur-in", "animate-text-blur-out");
      // Force reflow so the animation re-applies cleanly even if it just played.
      void el.offsetWidth;
      el.classList.add("animate-text-blur-out");
      const onEnd = () => {
        el.removeEventListener("animationend", onEnd);
        el.classList.remove("animate-text-blur-out");
        setPrompt("");
        resolve();
      };
      el.addEventListener("animationend", onEnd);
    });
  }, []);

  // Begin a streaming session: clear any previous bubbles, mark the time so
  // FloatingBubbles can ignore historical events, and persist to localStorage
  // so a refresh resumes the same session within the TTL.
  const beginSession = useCallback((name: string, namespace: string) => {
    const now = Date.now();
    const next: StreamingSession = {
      name,
      namespace,
      startedAt: now,
      expiresAt: now + SESSION_TTL_MS,
    };
    setSession(next);
    saveSession(next);
  }, []);

  async function sendToExisting(target: { name: string; namespace: string }) {
    setSubmitting(true);
    setError(null);
    try {
      await createAgent({
        name: target.name,
        namespace: target.namespace,
        instructions: prompt,
      });
      beginSession(target.name, target.namespace);
      blurOutPrompt();
    } catch (e) {
      setError(e instanceof Error ? e.message : "failed to send prompt");
      setSubmitting(false);
    }
  }

  async function createAndSend(namespace: string) {
    const name = nextManagerName(agents);
    setSubmitting(true);
    setError(null);
    try {
      const created = await createAgent({
        name,
        namespace,
        instructions: prompt,
        model: newModel,
        role: "manager",
        lifecycle: "Sleep",
        labels: { [PERSONAL_AGENT_LABEL]: "true" },
      });
      // Move out of "new" mode now that the agent exists, and treat the just-
      // created agent as the active selection for subsequent prompts.
      setAgents((prev) => [created, ...prev]);
      setActive(created);
      setIsNew(false);
      beginSession(created.name, created.namespace);
      blurOutPrompt();
    } catch (e) {
      setError(e instanceof Error ? e.message : "failed to create personal agent");
      setSubmitting(false);
    }
  }

  function handleGo() {
    if (!prompt.trim()) return;
    // Starting a new session blurs out the previous bubbles cleanly; the
    // FloatingBubbles component does that itself once `session` changes.
    if (isNew || !active) {
      createAndSend(newNamespace);
    } else {
      sendToExisting({ name: active.name, namespace: active.namespace });
    }
  }

  // Resume a session from localStorage on mount if one is still within TTL.
  useEffect(() => {
    const restored = loadSession();
    if (restored) setSession(restored);
  }, []);

  // Notify the parent whenever a session is/isn't active so the dashboard
  // can subtly blur the rest of the page while bubbles are visible.
  useEffect(() => {
    onSessionActiveChange?.(session !== null);
  }, [session, onSessionActiveChange]);

  // Called by FloatingBubbles when the agent's first thinking/text event
  // arrives. Flip the Go button back from spinner to "Go" so the user can
  // type the next prompt.
  const handleFirstResponse = useCallback(() => {
    setSubmitting(false);
  }, []);

  // Called by FloatingBubbles on task_completed/task_cancelled. The session
  // stays visible (bubbles persist) but localStorage is cleared so a refresh
  // doesn't try to resume a finished task.
  const handleTaskComplete = useCallback(() => {
    clearSession();
  }, []);

  // Called by FloatingBubbles when the user clicks the Clear bubble — drops
  // the session so the bubbles unmount with their exit animation. Agent
  // selection is preserved so the user can immediately type a new prompt.
  const handleClearBubbles = useCallback(() => {
    clearSession();
    setSession(null);
  }, []);

  // Namespace shown next to the agent picker:
  // - existing agent selected → its namespace, read-only
  // - new agent mode → editable, defaults to "default"
  const namespaceForDisplay = isNew || !active ? newNamespace : active.namespace;

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, delay: 0.05 }}
      className="mx-auto w-full max-w-2xl"
    >
      {/*
        The textarea takes the full container width so it sits visually
        centered; the Go button floats just past its right edge instead of
        shrinking the textarea on one side and leaving the other side wider.
      */}
      <div>
        {/* Selector row */}
        <div className="mb-2 flex flex-wrap items-center justify-center gap-2">
          <ActiveAgentChip
            active={active}
            isNew={isNew}
            agents={sorted}
            onSelect={handleSelectExisting}
            onSelectNew={handleSelectNew}
          />
          {/* Namespace — editable for new agent, static badge for existing.
              Crossfades between the two states so the swap isn't abrupt. */}
          <AnimatePresence mode="wait" initial={false}>
            <motion.div
              key={isNew || !active ? "ns-edit" : `ns-static-${namespaceForDisplay}`}
              initial={{ opacity: 0, y: -4 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 4 }}
              transition={{ duration: 0.16, ease: "easeOut" }}
            >
              {isNew || !active ? (
                <NamespaceChip value={newNamespace} onChange={setNewNamespace} />
              ) : (
                <span className="inline-flex items-center gap-1.5 rounded-full border border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 py-1 text-xs text-[var(--color-text-secondary)]">
                  <FolderOpen className="size-3 text-[var(--color-brand-blue)]" />
                  <span className="font-mono text-[var(--color-text)]">{namespaceForDisplay}</span>
                </span>
              )}
            </motion.div>
          </AnimatePresence>
          {/* Model — always editable. For a new agent the chip drives the
              `model` field of the create request. For an existing agent
              changing the value patches the CR so the next prompt runs on
              the chosen model. */}
          <ModelChip
            value={isNew || !active ? newModel : active.model || newModel}
            onChange={handleModelChange}
          />
          {error && <span className="text-xs text-red-400">{error}</span>}
        </div>
        {/* Textarea + Go button. The Go button floats absolutely just past
            the right edge of the textarea so the textarea itself stays
            full-width and visually centered in the container. */}
        <div className="relative">
          <textarea
            key={`textarea-${blurInToken}`}
            ref={textareaRef}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            onKeyDown={(e) => {
              // Enter submits, Shift+Enter inserts a newline. Cmd/Ctrl+Enter
              // also submits (kept for muscle memory from chat clients).
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                handleGo();
              }
            }}
            placeholder={session !== null ? "Enter another prompt for the agent..." : placeholder}
            rows={2}
            className={`w-full resize-none min-h-[3.25rem] max-h-[7rem] rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3.5 py-2.5 text-sm leading-relaxed text-[var(--color-text)] placeholder:text-[var(--color-text-muted)] shadow-[inset_0_1px_2px_rgba(var(--shadow-inset-color),0.15)] transition-[border-color,background-color,box-shadow,height] duration-150 hover:border-[var(--color-border-hover)] hover:bg-[var(--color-surface-hover)] focus:border-[var(--color-brand-blue)]/60 focus:bg-[var(--color-surface)] focus:shadow-[inset_0_1px_2px_rgba(var(--shadow-inset-color),0.15),0_0_0_3px_var(--color-brand-blue-glow)] focus:outline-none disabled:opacity-60 disabled:cursor-not-allowed${blurInToken > 0 ? " animate-text-blur-in" : ""}`}
            disabled={submitting}
          />
          <Button
            type="button"
            onClick={handleGo}
            disabled={!prompt.trim() || submitting}
            size="md"
            className="button-glint absolute left-full top-1/2 ml-3 -translate-y-1/2 min-w-[5.5rem]"
          >
            <AnimatePresence mode="wait" initial={false}>
              {submitting ? (
                <motion.span
                  key="busy"
                  initial={{ opacity: 0, scale: 0.7 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.7 }}
                  transition={{ duration: 0.18, ease: "easeOut" }}
                  className="inline-flex items-center"
                >
                  <Loader2 className="size-4 animate-spin" />
                </motion.span>
              ) : (
                <motion.span
                  key="idle"
                  initial={{ opacity: 0, scale: 0.7 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.7 }}
                  transition={{ duration: 0.18, ease: "easeOut" }}
                  className="inline-flex items-center gap-1.5"
                >
                  <Send className="size-3.5" />
                  <span>Go</span>
                </motion.span>
              )}
            </AnimatePresence>
          </Button>
        </div>

        {/*
          Streaming preview row. The bubbles render in an absolute layer
          starting at the row's top edge so they overlap the example pills
          beneath without affecting layout.
        */}
        <div className="relative h-0">
          <FloatingBubbles
            session={session}
            onFirstResponse={handleFirstResponse}
            onTaskComplete={handleTaskComplete}
            onClear={handleClearBubbles}
          />
        </div>
      </div>

      <div className="mt-2 flex flex-wrap items-center gap-1.5">
        {PILL_PROMPTS.map((p) => (
          <button
            key={p.label}
            type="button"
            onClick={() => {
              setPrompt(p.prompt);
              setBlurInToken((t) => t + 1);
            }}
            className="rounded-full border border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 py-0.5 text-[11px] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)] hover:text-[var(--color-text)] transition-colors cursor-pointer"
          >
            {p.label}
          </button>
        ))}
      </div>
    </motion.div>
  );
}
