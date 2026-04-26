"use client";

import { useState, useEffect, useRef } from "react";
import { Button } from "@/components/kit/button";
import { Input } from "@/components/kit/input";
import { Label } from "@/components/kit/label";
import { ChevronRight, Moon } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { listAgents, listSquads, createSquad, addSquadMember, patchAgent } from "@/lib/api";
import type { AgentResponse, Squad } from "@/lib/types";
import {
  AgentFieldsForm,
  type AgentFormValues,
  buildAgentSpecForSquad,
} from "./agent-fields-form";

const NAME_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/;

export interface TeamUpState {
  values: AgentFormValues;
  teamUpWithAgent: string; // "<namespace>/<name>"
  squadName: string;
}

export interface TeamUpModeFormProps {
  state: TeamUpState;
  onChange: (next: TeamUpState) => void;
  active: boolean;
  error: string | null;
  onError: (error: string | null) => void;
  onCreated?: () => void;
  onCancel: () => void;
}

export function TeamUpModeForm({ state, onChange, active, error, onError, onCreated, onCancel }: TeamUpModeFormProps) {
  const [availableAgents, setAvailableAgents] = useState<AgentResponse[]>([]);
  const [squads, setSquads] = useState<Squad[]>([]);
  const [agentSearch, setAgentSearch] = useState("");
  const [agentDropdownOpen, setAgentDropdownOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [sleepingPartner, setSleepingPartner] = useState(false);

  async function sleepPartner(name: string, namespace: string) {
    setSleepingPartner(true);
    onError(null);
    try {
      await patchAgent(name, { lifecycle: "Sleep" }, namespace);
      const res = await listAgents();
      setAvailableAgents(res.agents ?? []);
    } catch (e: unknown) {
      onError(e instanceof Error ? e.message : `Failed to sleep ${name}`);
    } finally {
      setSleepingPartner(false);
    }
  }
  const dropdownRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [dropdownPos, setDropdownPos] = useState<{ top: number; left: number; width: number; maxHeight: number } | null>(null);

  const { values, teamUpWithAgent, squadName } = state;

  useEffect(() => {
    if (!active) return;
    listAgents()
      .then((res) => setAvailableAgents(res.agents ?? []))
      .catch(() => setAvailableAgents([]));
    listSquads()
      .then((res) => setSquads(res.squads ?? []))
      .catch(() => setSquads([]));
  }, [active]);

  // Close dropdown when the form is no longer active (e.g. dialog closed or tab switched)
  useEffect(() => {
    if (!active) setAgentDropdownOpen(false);
  }, [active]);

  // Close dropdown on outside click
  useEffect(() => {
    if (!agentDropdownOpen) return;
    function onPointerDown(e: PointerEvent) {
      const target = e.target as Node;
      const inDropdown = dropdownRef.current?.contains(target);
      const inTrigger = triggerRef.current?.contains(target);
      if (!inDropdown && !inTrigger) {
        setAgentDropdownOpen(false);
      }
    }
    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, [agentDropdownOpen]);

  // Compute dropdown position when opening (and on window resize/scroll while open)
  useEffect(() => {
    if (!agentDropdownOpen) {
      setDropdownPos(null);
      return;
    }
    function reposition() {
      const trigger = triggerRef.current;
      if (!trigger) return;
      const rect = trigger.getBoundingClientRect();
      const margin = 8;
      const top = rect.bottom + 4;
      const maxHeight = Math.max(120, window.innerHeight - top - margin);
      setDropdownPos({ top, left: rect.left, width: rect.width, maxHeight });
    }
    reposition();
    window.addEventListener("resize", reposition);
    window.addEventListener("scroll", reposition, true);
    return () => {
      window.removeEventListener("resize", reposition);
      window.removeEventListener("scroll", reposition, true);
    };
  }, [agentDropdownOpen]);

  // Determine if the selected agent is already part of a squad
  const matchingSquad = (() => {
    if (!teamUpWithAgent) return null;
    const [agentNs, agentName] = teamUpWithAgent.split("/");
    return squads.find((s) =>
      s.namespace === agentNs && s.members?.some((m) => m.name === agentName)
    ) ?? null;
  })();

  const squadNameLocked = matchingSquad !== null;
  const noAgentSelected = !teamUpWithAgent;

  // Resolve the selected partner agent so we can preflight its phase. Only solo
  // (not-yet-in-a-squad) agents need to be asleep — agents already in a squad
  // are managed by the squad controller, so adding them to another squad isn't
  // a typical flow but doesn't trigger the dual-pod problem.
  const selectedPartnerAgent = teamUpWithAgent
    ? (() => {
        const [ns, n] = teamUpWithAgent.split("/");
        return availableAgents.find((a) => a.name === n && a.namespace === ns) ?? null;
      })()
    : null;
  const partnerMustBeAsleep =
    selectedPartnerAgent !== null &&
    !matchingSquad &&
    selectedPartnerAgent.status !== "Sleeping";

  // When matching squad changes, sync squadName
  useEffect(() => {
    if (matchingSquad) {
      if (state.squadName !== matchingSquad.name) {
        onChange({ ...state, squadName: matchingSquad.name });
      }
    }
    // When no squad match: leave whatever the user typed
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [matchingSquad?.name]);

  function validate(): string | null {
    if (!values.name.trim()) return "Name is required.";
    if (!NAME_PATTERN.test(values.name)) return "Name must be lowercase letters, numbers, and hyphens only.";
    if (!values.instructions.trim()) return "Instructions are required.";
    if (!teamUpWithAgent) return "Select an agent to team up with.";
    if (partnerMustBeAsleep && selectedPartnerAgent) {
      return `Agent "${selectedPartnerAgent.name}" must be asleep in order to team up. Sleep it first, then retry.`;
    }
    if (!squadName.trim()) return "Squad name is required.";
    if (!squadNameLocked && !NAME_PATTERN.test(squadName)) return "Squad name must be lowercase letters, numbers, and hyphens only.";
    return null;
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const validationError = validate();
    if (validationError) {
      onError(validationError);
      return;
    }

    setSubmitting(true);
    onError(null);

    try {
      const [agentNs, agentName] = teamUpWithAgent.split("/");
      const newAgentSpec = buildAgentSpecForSquad(values);
      const newAgentName = values.name.trim();

      if (matchingSquad) {
        await addSquadMember(matchingSquad.name, matchingSquad.namespace, {
          name: newAgentName,
          spec: newAgentSpec,
        });
      } else {
        await createSquad({
          name: squadName.trim(),
          namespace: values.namespace.trim() || undefined,
          members: [
            { ref: { name: agentName, namespace: agentNs } },
            { name: newAgentName, spec: newAgentSpec },
          ],
        });
      }

      onCreated?.();
    } catch (e: unknown) {
      onError(e instanceof Error ? e.message : "Failed to team up.");
    } finally {
      setSubmitting(false);
    }
  }

  const filteredAgents = availableAgents.filter((a) => {
    const q = agentSearch.toLowerCase();
    return !q || a.name.toLowerCase().includes(q) || a.namespace.toLowerCase().includes(q);
  });

  const selectedAgentDisplay = teamUpWithAgent
    ? (() => {
        const [ns, n] = teamUpWithAgent.split("/");
        const agent = availableAgents.find((a) => a.name === n && a.namespace === ns);
        return agent ? agent.name : teamUpWithAgent;
      })()
    : null;

  return (
    <form onSubmit={handleSubmit} className="flex flex-col flex-1 min-h-0">
      <div className="flex-1 min-h-0 overflow-y-auto pr-1 flex flex-col gap-4">
        {/* Full agent form */}
        <AgentFieldsForm
          values={values}
          onChange={(next) => onChange({ ...state, values: next })}
          active={active}
          idPrefix="teamup"
        />

        {/* Team Up section */}
        <div className="border-t border-[var(--color-border)] pt-4 flex flex-col gap-3">
          <span className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-muted)]">Team Up</span>

          {/* Agent picker */}
          <div className="flex flex-col gap-1.5">
            <Label>Team Up With</Label>
            <div className="relative">
              <button
                ref={triggerRef}
                type="button"
                onClick={() => setAgentDropdownOpen((v) => !v)}
                className="flex h-9 w-full items-center justify-between rounded-md border border-[var(--color-border)] bg-[var(--color-bg-subtle)] px-3 py-1 text-sm transition-colors hover:border-[var(--color-border-hover)] cursor-pointer"
              >
                {selectedAgentDisplay ? (
                  <span className="text-[var(--color-text)]">{selectedAgentDisplay}</span>
                ) : (
                  <span className="text-[var(--color-text-muted)]">Select an agent...</span>
                )}
                <ChevronRight className={`size-3.5 text-[var(--color-text-secondary)] transition-transform ${agentDropdownOpen ? "rotate-90" : ""}`} />
              </button>
              <AnimatePresence>
                {agentDropdownOpen && dropdownPos && (
                <motion.div
                  ref={dropdownRef}
                  style={{
                    position: "fixed",
                    top: dropdownPos.top,
                    left: dropdownPos.left,
                    width: dropdownPos.width,
                    maxHeight: dropdownPos.maxHeight,
                  }}
                  initial={{ opacity: 0, y: -4, scale: 0.98 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, y: -4, scale: 0.98 }}
                  transition={{ duration: 0.12, ease: "easeOut" }}
                  className="z-[100] rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-subtle)] shadow-[0_8px_32px_rgba(0,0,0,0.4),0_2px_8px_rgba(0,0,0,0.2)] flex flex-col">
                  <div className="p-1.5 border-b border-[var(--color-border)] shrink-0">
                    <input
                      autoFocus
                      type="text"
                      placeholder="Search agents..."
                      value={agentSearch}
                      onChange={(e) => setAgentSearch(e.target.value)}
                      className="w-full bg-transparent text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-muted)] outline-none px-1 py-0.5"
                    />
                  </div>
                  <div className="flex-1 min-h-0 overflow-y-auto">
                    {filteredAgents.length === 0 && (
                      <p className="px-3 py-2 text-xs text-[var(--color-text-muted)]">No agents found</p>
                    )}
                    {filteredAgents.map((agent) => {
                      const key = `${agent.namespace}/${agent.name}`;
                      const inSquad = squads.some(
                        (s) => s.namespace === agent.namespace && s.members?.some((m) => m.name === agent.name)
                      );
                      return (
                        <button
                          key={key}
                          type="button"
                          onClick={() => {
                            onChange({ ...state, teamUpWithAgent: key });
                            setAgentDropdownOpen(false);
                            setAgentSearch("");
                          }}
                          className={`flex w-full items-center gap-2 px-3 py-2 text-sm text-left hover:bg-[var(--color-surface-hover)] transition-colors cursor-pointer ${
                            teamUpWithAgent === key ? "text-[var(--color-text)]" : "text-[var(--color-text-secondary)]"
                          }`}
                        >
                          <span className="flex-1">{agent.name}</span>
                          <span className="text-[9px] text-[var(--color-brand-blue-light)]">{agent.namespace}</span>
                          {inSquad && (
                            <span className="text-[9px] px-1.5 py-0.5 rounded bg-[var(--color-brand-violet)]/10 text-[var(--color-brand-violet)]">squad</span>
                          )}
                        </button>
                      );
                    })}
                  </div>
                </motion.div>
                )}
              </AnimatePresence>
            </div>
          </div>

          {/* Squad name */}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="teamup-squad-name">
              Squad Name
              {squadNameLocked && (
                <span className="ml-2 text-[10px] text-[var(--color-brand-violet)] font-normal">
                  (using existing squad)
                </span>
              )}
              {noAgentSelected && (
                <span className="ml-2 text-[10px] text-[var(--color-text-muted)] font-normal">
                  (select an agent first)
                </span>
              )}
            </Label>
            <Input
              id="teamup-squad-name"
              placeholder={squadNameLocked ? "" : "my-squad"}
              value={squadName}
              onChange={(e) => {
                if (squadNameLocked || noAgentSelected) return;
                onChange({ ...state, squadName: e.target.value });
              }}
              disabled={squadNameLocked || noAgentSelected}
              className={squadNameLocked || noAgentSelected ? "opacity-60 cursor-not-allowed" : ""}
              autoComplete="off"
            />
          </div>
        </div>

        {partnerMustBeAsleep && selectedPartnerAgent && (
          <div className="rounded-[var(--radius-md)] border border-amber-500/40 bg-amber-500/5 p-3 text-sm text-amber-300/90 space-y-2">
            <p>
              Agent <span className="font-medium text-amber-200">{selectedPartnerAgent.name}</span> must be asleep before teaming up.
            </p>
            <div className="flex items-center justify-between gap-2 rounded bg-amber-500/5 px-2 py-1">
              <span className="text-sm">
                <span className="font-medium text-amber-200">{selectedPartnerAgent.name}</span>
                <span className="ml-1.5 text-xs text-amber-300/70">({selectedPartnerAgent.status})</span>
              </span>
              <Button
                type="button"
                size="sm"
                variant="secondary"
                onClick={() => sleepPartner(selectedPartnerAgent.name, selectedPartnerAgent.namespace)}
                disabled={sleepingPartner}
              >
                <Moon className={`size-3 ${sleepingPartner ? "animate-pulse" : ""}`} data-icon="inline-start" />
                {sleepingPartner ? "Sleeping…" : "Sleep"}
              </Button>
            </div>
          </div>
        )}

        {error && <p className="text-sm text-red-400">{error}</p>}
      </div>

      <div className="mt-4 shrink-0 flex justify-end gap-2">
        <Button variant="secondary" type="button" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={submitting || partnerMustBeAsleep}>
          {submitting ? "Creating..." : matchingSquad ? "Add to Squad" : "Team Up"}
        </Button>
      </div>
    </form>
  );
}
