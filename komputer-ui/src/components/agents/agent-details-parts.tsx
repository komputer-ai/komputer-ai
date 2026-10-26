"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { AnimatePresence, motion } from "framer-motion";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Maximize2, X, Copy, Check, Loader2, Lock } from "lucide-react";
import { Button } from "@/components/kit/button";
import { Textarea } from "@/components/kit/textarea";
import { Tooltip } from "@/components/kit/tooltip";
import { cn } from "@/lib/utils";

// One full-bleed row of the agent details form: title and description on the
// left, the fields across the rest. Rows are separated by a divider, not cards.
export function DetailsSection({
  icon: Icon,
  title,
  description,
  dirty,
  index = 0,
  children,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
  dirty?: boolean;
  /** Position in the form, for a short staggered entrance. */
  index?: number;
  children: React.ReactNode;
}) {
  return (
    <motion.section
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: "easeOut", delay: index * 0.04 }}
      className="group/section grid grid-cols-[220px_minmax(0,1fr)] @max-3xl:grid-cols-1 gap-x-7 gap-y-3 px-7 py-5 border-b border-[var(--color-border)] transition-colors duration-150 hover:bg-[var(--color-surface)]/40"
    >
      <div>
        <h2 className="flex items-center gap-2 text-sm font-semibold text-[var(--color-text)]">
          <Icon className="size-3.5 text-[var(--color-text-secondary)] transition-colors group-hover/section:text-[var(--color-brand-blue-light)]" />
          {title}
          <AnimatePresence>
            {dirty && (
              <Tooltip side="right" content="Unsaved changes in this section">
                <motion.span
                  initial={{ scale: 0 }}
                  animate={{ scale: 1 }}
                  exit={{ scale: 0 }}
                  className="block size-1.5 rounded-full bg-[var(--color-brand-blue)]"
                  aria-label="Unsaved changes"
                />
              </Tooltip>
            )}
          </AnimatePresence>
        </h2>
        <p className="mt-1 text-[12.5px] leading-relaxed text-[var(--color-text)]/70">{description}</p>
      </div>
      <div className="grid min-w-0 gap-4">{children}</div>
    </motion.section>
  );
}

// A section of the details rail: a flush block with a small uppercase heading.
export function RailSection({ title, action, children }: { title: string; action?: React.ReactNode; children: React.ReactNode }) {
  return (
    <section className="flex flex-col gap-2.5 px-5 py-4 border-b border-[var(--color-border)]">
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text)]">{title}</h3>
        {action}
      </div>
      {children}
    </section>
  );
}

// A field's label plus a one-line hint under its control.
export function Field({ label, tip, hint, hintTone, className, children }: {
  label: string;
  /** What the field does, shown on hovering its label. */
  tip?: string;
  hint?: React.ReactNode;
  hintTone?: "next" | "changed";
  className?: string;
  children: React.ReactNode;
}) {
  const labelEl = (
    <span
      className={cn(
        "text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text)]",
        tip && "cursor-help underline decoration-dotted decoration-[var(--color-text-muted)] underline-offset-4",
      )}
    >
      {label}
    </span>
  );
  return (
    <div className={cn("flex min-w-0 flex-col gap-1.5", className)}>
      {tip ? <span className="self-start"><Tooltip side="top" align="start" content={tip}>{labelEl}</Tooltip></span> : labelEl}
      {children}
      {hint && (
        <span
          className={cn(
            "text-[11.5px] leading-snug",
            hintTone === "next" ? "text-[var(--color-status-sleeping)]" : hintTone === "changed" ? "text-[var(--color-brand-blue-light)]" : "text-[var(--color-text)]/65",
          )}
        >
          {hint}
        </span>
      )}
    </div>
  );
}

// "in 28m", "in 23h", "in 29d" for a future timestamp; "" when it's not in the future.
export function fmtIn(at?: string): string {
  if (!at) return "";
  const ms = new Date(at).getTime() - Date.now();
  if (Number.isNaN(ms) || ms <= 0) return "";
  const m = Math.round(ms / 60_000);
  if (m < 60) return `in ${Math.max(m, 1)}m`;
  const h = Math.round(m / 60);
  if (h < 48) return `in ${h}h`;
  return `in ${Math.round(h / 24)}d`;
}

function textMeta(s: string): string {
  if (!s.trim()) return "";
  const lines = s.split("\n").length;
  return lines > 1 ? `${lines} lines · ${s.length} chars` : `${s.length} chars`;
}

type PromptTab = "instructions" | "system";

// Instructions and the system prompt as two tabs over one full-width editor.
// Instructions are read-only here (they come from the last task); the system
// prompt is editable and opens in a focus editor with a Markdown preview.
const INSTRUCTIONS_READ_ONLY = "Read-only: these are the last task's instructions. Send a new task from the chat to change them.";

export function PromptEditor({ instructions, systemPrompt, onSystemPromptChange }: {
  instructions: string;
  systemPrompt: string;
  onSystemPromptChange: (v: string) => void;
}) {
  const [tab, setTab] = useState<PromptTab>("instructions");
  const [focusOpen, setFocusOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  // Where the read-only tooltip follows the mouse over the instructions pane.
  const [cursorTip, setCursorTip] = useState<{ x: number; y: number } | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const current = tab === "instructions" ? instructions : systemPrompt;

  const openFocus = useCallback(() => {
    setTab("system");
    setFocusOpen(true);
  }, []);

  // Cmd/Ctrl+Shift+E opens the system prompt in the focus editor.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === "e") {
        e.preventDefault();
        openFocus();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [openFocus]);

  async function copy() {
    try {
      await navigator.clipboard.writeText(current);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // Clipboard can be unavailable (insecure context); copying is best-effort.
    }
  }

  const tabBtn = (id: PromptTab, label: string, value: string, locked?: boolean) => (
    <button
      type="button"
      role="tab"
      aria-selected={tab === id}
      onClick={() => setTab(id)}
      className={cn(
        "relative inline-flex items-center gap-2 whitespace-nowrap px-3 py-2 text-[12.5px] transition-colors cursor-pointer",
        tab === id ? "font-medium text-[var(--color-text)]" : "text-[var(--color-text-secondary)] hover:text-[var(--color-text)]",
      )}
    >
      {locked && <Lock className="size-3 text-[var(--color-text-secondary)]" aria-label="Read-only" />}
      {label}
      {value.trim() && <span className="font-mono text-[11px] font-normal text-[var(--color-text-secondary)]">{textMeta(value)}</span>}
      {tab === id && (
        <motion.span
          layoutId="prompt-tab-underline"
          className="absolute inset-x-1 -bottom-px h-0.5 rounded-full bg-[var(--color-brand-blue)]"
          transition={{ type: "spring", stiffness: 500, damping: 38 }}
        />
      )}
    </button>
  );

  return (
    <div className="rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-bg-subtle)]">
      <div className="flex items-center gap-1 border-b border-[var(--color-border)] px-1.5" role="tablist" aria-label="Prompt">
        {tabBtn("instructions", "Instructions", instructions, true)}
        {tabBtn("system", "System prompt", systemPrompt)}
        <span className="ml-auto inline-flex gap-0.5">
          <Tooltip side="bottom" content={copied ? "Copied" : "Copy"}>
            <button type="button" onClick={copy} disabled={!current} aria-label="Copy" className="inline-flex size-7 items-center justify-center rounded-[var(--radius-sm)] text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-hover)] disabled:opacity-40 cursor-pointer">
              {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
            </button>
          </Tooltip>
          <Tooltip side="bottom" content="Edit the system prompt full screen (⌘⇧E)">
            <button type="button" onClick={openFocus} aria-label="Open focus editor" className="inline-flex size-7 items-center justify-center rounded-[var(--radius-sm)] text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-hover)] cursor-pointer">
              <Maximize2 className="size-3.5" />
            </button>
          </Tooltip>
        </span>
      </div>

      {/* One editor area for both tabs: a fixed height (drag the bottom edge to
          resize) so switching tabs never changes its size; the content crossfades. */}
      <div className="relative h-[240px] min-h-[140px] max-h-[70vh] resize-y overflow-hidden">
        <AnimatePresence mode="wait" initial={false}>
          <motion.div
            key={tab}
            className="absolute inset-0"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.12, ease: "easeOut" }}
          >
            {tab === "instructions" ? (
              <div
                aria-readonly="true"
                aria-disabled="true"
                aria-label={INSTRUCTIONS_READ_ONLY}
                onMouseMove={(e) => setCursorTip({ x: e.clientX, y: e.clientY })}
                onMouseLeave={() => setCursorTip(null)}
                className="h-full cursor-not-allowed overflow-y-auto whitespace-pre-wrap px-3 py-2.5 font-mono text-[12.5px] leading-relaxed text-[var(--color-text)]/60"
              >
                {instructions || "No instructions yet. Send the agent a task from the chat."}
              </div>
            ) : (
              <Textarea
                ref={textareaRef}
                value={systemPrompt}
                onChange={(e) => onSystemPromptChange(e.target.value)}
                placeholder="Persona, rules or output format the agent keeps on every task. Markdown is supported."
                className="h-full min-h-0 resize-none rounded-none border-0 bg-transparent shadow-none focus:shadow-none"
              />
            )}
          </motion.div>
        </AnimatePresence>
      </div>

      {cursorTip && tab === "instructions" && typeof document !== "undefined" &&
        createPortal(
          <div
            className="pointer-events-none fixed z-[60] w-max max-w-[300px] rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface-raised)] px-2.5 py-1 text-[11px] font-medium text-[var(--color-text)] shadow-[0_4px_12px_rgba(0,0,0,0.3)]"
            style={{ left: Math.min(cursorTip.x + 14, window.innerWidth - 316), top: cursorTip.y + 18 }}
            role="tooltip"
          >
            {INSTRUCTIONS_READ_ONLY}
          </div>,
          document.body,
        )}

      <FocusEditor
        open={focusOpen}
        value={systemPrompt}
        onChange={onSystemPromptChange}
        onClose={() => setFocusOpen(false)}
      />
    </div>
  );
}

// Full-screen editor for a long system prompt, with a live Markdown preview.
// Opens and closes like the kit Dialog: a blurred backdrop fades in and the
// panel scales and slides up, rendered in a portal above the page.
function FocusEditor({ open, value, onChange, onClose }: { open: boolean; value: string; onChange: (v: string) => void; onClose: () => void }) {
  if (typeof document === "undefined") return null;
  return createPortal(
    <AnimatePresence>
      {open && <FocusEditorPanel key="focus-editor" value={value} onChange={onChange} onClose={onClose} />}
    </AnimatePresence>,
    document.body,
  );
}

function FocusEditorPanel({ value, onChange, onClose }: { value: string; onChange: (v: string) => void; onClose: () => void }) {
  // The draft starts from the current prompt each time the editor opens.
  const [draft, setDraft] = useState(value);
  const ref = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    ref.current?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      window.removeEventListener("keydown", onKey);
      document.body.style.overflow = prev;
    };
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-6" role="dialog" aria-modal="true" aria-label="System prompt editor">
      <motion.div
        className="absolute inset-0 bg-[rgba(10,5,20,0.65)] backdrop-blur-md"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: 0.15 }}
        onMouseDown={onClose}
      />
      <motion.div
        className="relative flex h-full max-h-[860px] w-full max-w-[1280px] flex-col rounded-[var(--radius-lg)] border border-[var(--color-border-hover)] bg-[var(--color-surface)] shadow-[0_24px_64px_rgba(0,0,0,0.45)]"
        initial={{ opacity: 0, scale: 0.95, y: 10 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        exit={{ opacity: 0, scale: 0.95, y: 10 }}
        transition={{ duration: 0.15, ease: "easeOut" }}
      >
        <div className="flex items-center gap-3 border-b border-[var(--color-border)] px-4 py-2.5">
          <span className="text-sm font-semibold text-[var(--color-text)]">System prompt</span>
          <span className="font-mono text-[11.5px] text-[var(--color-text-secondary)]">{textMeta(draft)}</span>
          <Tooltip side="bottom" align="end" content="Close (Esc)">
            <button type="button" onClick={onClose} aria-label="Close" className="ml-auto inline-flex size-7 items-center justify-center rounded-[var(--radius-sm)] text-[var(--color-text-secondary)] transition-colors hover:bg-[var(--color-surface-hover)] hover:text-[var(--color-text)] cursor-pointer">
              <X className="size-4" />
            </button>
          </Tooltip>
        </div>
        <div className="grid min-h-0 flex-1 grid-cols-2">
          <div className="flex min-h-0 flex-col">
            <span className="px-4 pt-3 text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text)]">Edit</span>
            <textarea
              ref={ref}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="Persona, rules or output format the agent keeps on every task."
              className="min-h-0 flex-1 resize-none bg-transparent px-4 py-2 font-mono text-[13px] leading-relaxed text-[var(--color-text)] placeholder:text-[var(--color-text-muted)] focus:outline-none"
            />
          </div>
          <div className="flex min-h-0 flex-col border-l border-[var(--color-border)]">
            <span className="px-4 pt-3 text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text)]">Preview</span>
            <div className="prose-chat min-h-0 flex-1 overflow-y-auto px-4 py-2 text-sm text-[var(--color-text)]">
              {draft.trim() ? <ReactMarkdown remarkPlugins={[remarkGfm]}>{draft}</ReactMarkdown> : <span className="text-[var(--color-text)]/60">Nothing to preview yet.</span>}
            </div>
          </div>
        </div>
        <div className="flex justify-end gap-2 border-t border-[var(--color-border)] px-4 py-2.5">
          <Button size="sm" variant="secondary" onClick={onClose}>Cancel</Button>
          <Button
            size="sm"
            variant="primary"
            onClick={() => {
              onChange(draft);
              onClose();
            }}
          >
            Done
          </Button>
        </div>
      </motion.div>
    </div>
  );
}

// A flush strip at the bottom of the details form, shown only with unsaved
// edits (or while a save is finishing or failed).
export function SaveBar({ changed, saving, saved, error, disabled, onDiscard, onSave }: {
  changed: string[];
  saving: boolean;
  saved: boolean;
  error: string | null;
  disabled?: boolean;
  onDiscard: () => void;
  onSave: () => void;
}) {
  const visible = changed.length > 0 || saving || saved || !!error;
  return (
    <AnimatePresence>
      {visible && (
    <motion.div
      initial={{ y: 24, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      exit={{ y: 24, opacity: 0 }}
      transition={{ duration: 0.18, ease: "easeOut" }}
      className="sticky bottom-0 z-10 mt-auto flex items-center gap-3 border-t border-[var(--color-border-hover)] bg-[var(--color-surface)] px-7 py-2.5 text-[13px]"
      role="status"
    >
      {error ? (
        <span className="min-w-0 truncate text-[var(--color-status-error)]" title={error}>{error}</span>
      ) : saved && changed.length === 0 ? (
        <span className="inline-flex items-center gap-1.5 text-[var(--color-status-running)]"><Check className="size-3.5" />Saved</span>
      ) : (
        <>
          <span className="size-2 rounded-full bg-[var(--color-brand-blue)]" aria-hidden />
          <span className="text-[var(--color-text)]">{changed.length} unsaved {changed.length === 1 ? "change" : "changes"}</span>
          <span className="min-w-0 truncate text-[var(--color-text)]/80">{changed.join(", ")}</span>
        </>
      )}
      <span className="flex-1" />
      {changed.length > 0 && (
        <>
          <Button size="sm" variant="secondary" onClick={onDiscard} disabled={saving}>Discard</Button>
          <Button size="sm" variant="primary" onClick={onSave} disabled={disabled || saving}>
            {saving ? <><Loader2 className="size-3 animate-spin" data-icon="inline-start" />Saving…</> : "Save changes"}
          </Button>
        </>
      )}
    </motion.div>
      )}
    </AnimatePresence>
  );
}
