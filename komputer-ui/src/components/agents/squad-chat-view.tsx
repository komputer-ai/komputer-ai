"use client";

import { type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { AnimatePresence, motion } from "framer-motion";
import { Rows2, Columns2 } from "lucide-react";
import { PeerAgentPane } from "@/components/agents/peer-agent-pane";

const MAX_PANES = 3;
const PREF_TTL_MS = 30 * 24 * 60 * 60 * 1000; // 30 days

interface ChatModePref {
  multi: boolean;
  panes: string[];
  expiresAt: number;
}

function loadPref(squadName: string): ChatModePref | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(`squad-chat-mode:${squadName}`);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as ChatModePref;
    if (typeof parsed?.expiresAt !== "number" || parsed.expiresAt < Date.now()) {
      localStorage.removeItem(`squad-chat-mode:${squadName}`);
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

function savePref(squadName: string, multi: boolean, panes: string[]): void {
  if (typeof window === "undefined") return;
  try {
    const pref: ChatModePref = { multi, panes, expiresAt: Date.now() + PREF_TTL_MS };
    localStorage.setItem(`squad-chat-mode:${squadName}`, JSON.stringify(pref));
  } catch {
    // localStorage unavailable / quota — silently ignore
  }
}

export interface SquadChatViewMember {
  name: string;
  namespace: string;
}

export interface SquadChatViewProps {
  /** Squad name — used as the local-storage key for the user's mode preference. */
  squadName: string;
  /** Squad members in tab order. Caller passes only when squad has >1 members. */
  members: SquadChatViewMember[];
  /** The agent whose URL we're on — always pane 1, can't be deselected. */
  primaryAgentName: string;
  /** Already-wired AgentChat for the primary agent (rendered as pane 1). */
  primaryChat: ReactNode;
}

/**
 * Wraps the squad-tab row + a toggle for single/multi chat mode + N parallel
 * agent chat panes. The primary (URL) agent is always pane 1 and is rendered
 * via the `primaryChat` slot so its full event/websocket wiring stays on the
 * page. Additional panes are PeerAgentPane instances.
 *
 * Multi-mode selection persists in `?panes=alice,bob` so reload/share works.
 */
export function SquadChatView({ squadName, members, primaryAgentName, primaryChat }: SquadChatViewProps) {
  const searchParams = useSearchParams();

  // Resolve initial state once on mount: URL ?panes= wins (so shared links work);
  // otherwise fall back to the user's saved preference for this squad; otherwise
  // start in single mode with no extra panes.
  const initialStateRef = useRef<{ multi: boolean; panes: string[] } | null>(null);
  if (initialStateRef.current === null) {
    const urlPanes = parsePanesParam(searchParams?.get("panes"), members, primaryAgentName);
    if (urlPanes.length > 0 || searchParams?.get("panes") === "") {
      initialStateRef.current = { multi: urlPanes.length > 0, panes: urlPanes };
    } else {
      const pref = loadPref(squadName);
      if (pref) {
        const validPanes = pref.panes.filter(
          (n) => members.some((m) => m.name === n) && n !== primaryAgentName,
        );
        initialStateRef.current = { multi: pref.multi, panes: validPanes.slice(0, MAX_PANES - 1) };
      } else {
        initialStateRef.current = { multi: false, panes: [] };
      }
    }
  }
  const [rawExtraPanes, setRawExtraPanes] = useState<string[]>(initialStateRef.current.panes);
  const [multi, setMulti] = useState<boolean>(initialStateRef.current.multi);
  const [capWarning, setCapWarning] = useState(false);
  const capWarningTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Derive valid extra panes from raw state — drops references to members that
  // are no longer in the squad without needing a sync effect.
  const validNames = useMemo(() => new Set(members.map((m) => m.name)), [members]);
  const extraPanes = useMemo(
    () => rawExtraPanes.filter((n) => validNames.has(n) && n !== primaryAgentName),
    [rawExtraPanes, validNames, primaryAgentName]
  );

  // Push pane selection to the URL. Called only from user actions, never from
  // an effect — writing the URL on every render races with Next's router during
  // tab navigations and freezes the UI.
  const writeUrl = useCallback((panes: string[]) => {
    const params = new URLSearchParams(window.location.search);
    if (panes.length === 0) {
      params.delete("panes");
    } else {
      params.set("panes", panes.join(","));
    }
    const qs = params.toString();
    window.history.replaceState(null, "", `${window.location.pathname}${qs ? `?${qs}` : ""}`);
  }, []);

  // Persist mode + panes to local storage so the next visit to this squad
  // restores them, scoped per squadName with a 30-day TTL.
  const persist = useCallback(
    (nextMulti: boolean, nextPanes: string[]) => {
      savePref(squadName, nextMulti, nextPanes);
    },
    [squadName],
  );

  // If we restored panes from localStorage but the URL doesn't reflect them,
  // sync the URL once on mount so refresh stays consistent.
  const didInitialUrlSyncRef = useRef(false);
  useEffect(() => {
    if (didInitialUrlSyncRef.current) return;
    didInitialUrlSyncRef.current = true;
    if (rawExtraPanes.length > 0 && !searchParams?.get("panes")) {
      writeUrl(rawExtraPanes);
    }
  }, [rawExtraPanes, searchParams, writeUrl]);

  const flashCapWarning = useCallback(() => {
    setCapWarning(true);
    if (capWarningTimer.current) clearTimeout(capWarningTimer.current);
    capWarningTimer.current = setTimeout(() => setCapWarning(false), 2000);
  }, []);

  useEffect(() => {
    return () => {
      if (capWarningTimer.current) clearTimeout(capWarningTimer.current);
    };
  }, []);

  const togglePane = useCallback(
    (memberName: string) => {
      if (memberName === primaryAgentName) return; // primary can't be deselected
      const cur = extraPanes;
      let next: string[];
      if (cur.includes(memberName)) {
        next = cur.filter((n) => n !== memberName);
      } else if (cur.length + 1 >= MAX_PANES) {
        // 1 (primary) + extras would exceed cap
        flashCapWarning();
        return;
      } else {
        next = [...cur, memberName];
      }
      setRawExtraPanes(next);
      writeUrl(next);
      persist(true, next);
    },
    [primaryAgentName, extraPanes, flashCapWarning, writeUrl, persist]
  );

  const removePane = useCallback(
    (memberName: string) => {
      const next = extraPanes.filter((n) => n !== memberName);
      setRawExtraPanes(next);
      writeUrl(next);
      persist(multi, next);
    },
    [extraPanes, writeUrl, persist, multi]
  );

  const setMode = useCallback(
    (next: boolean) => {
      setMulti(next);
      if (!next) {
        setRawExtraPanes([]);
        writeUrl([]);
        persist(false, []);
      } else {
        persist(true, extraPanes);
      }
    },
    [writeUrl, persist, extraPanes]
  );

  const activeNames = useMemo(
    () => new Set([primaryAgentName, ...extraPanes]),
    [primaryAgentName, extraPanes]
  );

  // Map member name -> namespace for PeerAgentPane lookup
  const memberByName = useMemo(() => {
    const m = new Map<string, SquadChatViewMember>();
    for (const member of members) m.set(member.name, member);
    return m;
  }, [members]);

  return (
    <div className="flex flex-1 min-h-0 min-w-0 flex-col">
      {/* Squad tab row */}
      <div className="shrink-0 flex items-center gap-0.5 px-4 pt-2 border-b border-[var(--color-border)]">
        <div className="flex items-center gap-0.5 min-w-0 flex-1 overflow-x-auto">
        {members.map((member) => {
          const isActive = activeNames.has(member.name);
          const isPrimary = member.name === primaryAgentName;

          if (multi) {
            return (
              <button
                key={member.name}
                type="button"
                onClick={() => togglePane(member.name)}
                disabled={isPrimary}
                className={`relative px-3 py-2 text-sm font-medium transition-colors pb-[calc(0.5rem+3px)] ${
                  isActive
                    ? "text-[var(--color-text)] cursor-default"
                    : "text-[var(--color-text-secondary)] hover:text-[var(--color-text)] cursor-pointer"
                } ${isPrimary ? "cursor-default" : ""}`}
              >
                {member.name}
                {isActive && (
                  <span className="absolute bottom-0 left-0 right-0 h-[3px] bg-[var(--color-brand-blue)] rounded-full" />
                )}
              </button>
            );
          }

          // Single-mode: link navigation
          if (isPrimary) {
            return (
              <span
                key={member.name}
                className="relative px-3 py-2 text-sm font-medium text-[var(--color-text)] cursor-default pb-[calc(0.5rem+3px)]"
              >
                {member.name}
                <span className="absolute bottom-0 left-0 right-0 h-[3px] bg-[var(--color-brand-blue)] rounded-full" />
              </span>
            );
          }
          return (
            <Link
              key={member.name}
              href={`/agents/${member.name}?namespace=${member.namespace}`}
              className="px-3 py-2 text-sm font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors"
            >
              {member.name}
            </Link>
          );
        })}
        </div>

        {/* Right-side controls */}
        <div className="ml-2 shrink-0 flex items-center gap-2 pb-1">
          <AnimatePresence>
            {capWarning && (
              <motion.span
                initial={{ opacity: 0, x: 4 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.15 }}
                className="text-[11px] text-amber-400"
              >
                max {MAX_PANES} panes
              </motion.span>
            )}
          </AnimatePresence>
          {multi && (
            <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text-muted)]">
              {1 + extraPanes.length} / {MAX_PANES}
            </span>
          )}
          <div
            role="radiogroup"
            aria-label="Chat mode"
            className="relative flex h-7 items-center rounded-full border border-[var(--color-border)] bg-[var(--color-bg)] p-0.5 shadow-[inset_0_2px_4px_rgba(0,0,0,0.2)]"
          >
            {/* Sliding pill indicator */}
            <motion.div
              aria-hidden="true"
              className="absolute top-0.5 bottom-0.5 w-[calc(50%-2px)] rounded-full bg-[var(--color-brand-blue)]/15 border border-[var(--color-brand-blue)]/50"
              animate={{ left: multi ? "calc(50%)" : "2px" }}
              transition={{ type: "spring", stiffness: 500, damping: 35 }}
            />
            <button
              type="button"
              role="radio"
              aria-checked={!multi}
              onClick={() => setMode(false)}
              className={`relative z-10 flex h-full items-center gap-1 rounded-full px-2.5 text-[11px] font-medium transition-colors cursor-pointer ${
                !multi
                  ? "text-[var(--color-brand-blue)]"
                  : "text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
              }`}
            >
              <Rows2 className="size-3" />
              Single
            </button>
            <button
              type="button"
              role="radio"
              aria-checked={multi}
              onClick={() => setMode(true)}
              className={`relative z-10 flex h-full items-center gap-1 rounded-full px-2.5 text-[11px] font-medium transition-colors cursor-pointer ${
                multi
                  ? "text-[var(--color-brand-blue)]"
                  : "text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
              }`}
            >
              <Columns2 className="size-3" />
              Multi
            </button>
          </div>
        </div>
      </div>

      {/* Panes — equal-width columns side by side. Each pane manages its own
          internal vertical scroll; the row never overflows horizontally because
          flex-1 + min-w-0 lets every pane shrink to share width. */}
      <div className="flex flex-1 min-h-0 overflow-hidden">
        <div className="flex flex-1 min-w-0">{primaryChat}</div>
        {extraPanes.map((name) => {
          const member = memberByName.get(name);
          if (!member) return null;
          return (
            <PeerAgentPane
              key={name}
              agentName={member.name}
              agentNamespace={member.namespace}
              onRemove={() => removePane(name)}
            />
          );
        })}
      </div>
    </div>
  );
}

function parsePanesParam(
  raw: string | null | undefined,
  members: SquadChatViewMember[],
  primary: string,
): string[] {
  if (!raw) return [];
  const valid = new Set(members.map((m) => m.name));
  return raw
    .split(",")
    .map((s) => s.trim())
    .filter((n) => n && n !== primary && valid.has(n))
    .slice(0, MAX_PANES - 1);
}
