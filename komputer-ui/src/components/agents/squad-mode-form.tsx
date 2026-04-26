"use client";

import { useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { Button } from "@/components/kit/button";
import { Input } from "@/components/kit/input";
import { Label } from "@/components/kit/label";
import { Plus, X } from "lucide-react";
import { NamespaceSelector } from "@/components/shared/namespace-selector";
import { createSquad } from "@/lib/api";
import type { CreateSquadRequest } from "@/lib/types";
import {
  AgentFieldsForm,
  type AgentFormValues,
  makeDefaultAgentFormValues,
  buildAgentSpecForSquad,
} from "./agent-fields-form";

const NAME_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/;

export interface SquadState {
  squadName: string;
  namespace: string;
  activeSubtab: number;
  agents: AgentFormValues[];
}

export interface SquadModeFormProps {
  state: SquadState;
  onChange: (next: SquadState) => void;
  active: boolean;
  error: string | null;
  onError: (error: string | null) => void;
  onCreated?: () => void;
  onCancel: () => void;
}

export function SquadModeForm({ state, onChange, active, error, onError, onCreated, onCancel }: SquadModeFormProps) {
  const [submitting, setSubmitting] = useState(false);
  const [subtabErrors, setSubtabErrors] = useState<Record<number, string>>({});

  const { squadName, namespace, activeSubtab, agents } = state;

  function patchAgent(idx: number, next: AgentFormValues) {
    const nextAgents = agents.map((a, i) => (i === idx ? next : a));
    onChange({ ...state, agents: nextAgents });
  }

  function addAgent() {
    onChange({
      ...state,
      agents: [...agents, makeDefaultAgentFormValues({ namespace })],
      activeSubtab: agents.length,
    });
  }

  function removeAgent(idx: number) {
    if (agents.length <= 1) return;
    const nextAgents = agents.filter((_, i) => i !== idx);
    onChange({
      ...state,
      agents: nextAgents,
      activeSubtab: Math.min(activeSubtab, nextAgents.length - 1),
    });
  }

  function validate(): string | null {
    if (!squadName.trim()) return "Squad name is required.";
    if (!NAME_PATTERN.test(squadName)) return "Squad name must be lowercase letters, numbers, and hyphens only.";
    if (agents.length < 1) return "At least one agent is required.";

    const errors: Record<number, string> = {};
    const seen = new Set<string>();
    agents.forEach((agent, idx) => {
      const name = agent.name.trim();
      if (!name) {
        errors[idx] = "Agent name is required.";
        return;
      }
      if (!NAME_PATTERN.test(name)) {
        errors[idx] = "Agent name must be lowercase letters, numbers, and hyphens only.";
        return;
      }
      if (seen.has(name)) {
        errors[idx] = `Duplicate name "${name}".`;
        return;
      }
      seen.add(name);
      if (!agent.instructions.trim()) {
        errors[idx] = "Instructions are required.";
      }
    });
    if (Object.keys(errors).length > 0) {
      setSubtabErrors(errors);
      return "Fix the errors in each agent tab before submitting.";
    }
    setSubtabErrors({});
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
      const members: CreateSquadRequest["members"] = agents.map((agent) => ({
        name: agent.name.trim(),
        spec: buildAgentSpecForSquad(agent),
      }));

      const req: CreateSquadRequest = {
        name: squadName.trim(),
        namespace: namespace.trim() || undefined,
        members,
      };

      await createSquad(req);
      onCreated?.();
    } catch (e: unknown) {
      onError(e instanceof Error ? e.message : "Failed to create squad.");
    } finally {
      setSubmitting(false);
    }
  }

  const activeAgent = agents[activeSubtab] ?? agents[0];

  return (
    <form onSubmit={handleSubmit} className="flex flex-col flex-1 min-h-0">
      <div className="flex-1 min-h-0 overflow-y-auto pr-1 flex flex-col gap-4">
        {/* Squad name + namespace */}
        <div className="grid grid-cols-2 gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="squad-name">Squad Name</Label>
            <Input
              id="squad-name"
              placeholder="my-squad"
              value={squadName}
              onChange={(e) => onChange({ ...state, squadName: e.target.value })}
              autoComplete="off"
            />
          </div>
          <NamespaceSelector value={namespace} onChange={(v) => onChange({ ...state, namespace: v })} />
        </div>

        {/* Agent subtabs */}
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-1 border-b border-[var(--color-border)] pb-px overflow-x-auto">
            {agents.map((agent, idx) => (
              <div key={idx} className="relative flex items-center shrink-0">
                <button
                  type="button"
                  onClick={() => onChange({ ...state, activeSubtab: idx })}
                  className={`relative px-3 py-2 text-sm font-medium transition-colors cursor-pointer ${
                    activeSubtab === idx
                      ? "text-[var(--color-text)]"
                      : "text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
                  }`}
                >
                  {agent.name.trim() || `Agent ${idx + 1}`}
                  {subtabErrors[idx] && (
                    <span className="ml-1 inline-block size-1.5 rounded-full bg-red-400 align-middle" />
                  )}
                  {activeSubtab === idx && (
                    <motion.div
                      className="absolute bottom-0 left-0 right-0 h-[3px] bg-[var(--color-brand-blue)] rounded-full shadow-[0_1px_4px_var(--color-brand-blue-glow)]"
                      layoutId="squad-subtab-indicator"
                      transition={{ duration: 0.2, ease: "easeInOut" }}
                    />
                  )}
                </button>
                {agents.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeAgent(idx)}
                    className="ml-0.5 p-0.5 text-[var(--color-text-muted)] hover:text-red-400 transition-colors cursor-pointer"
                  >
                    <X className="size-3" />
                  </button>
                )}
              </div>
            ))}
            <button
              type="button"
              onClick={addAgent}
              className="flex items-center gap-1 ml-1 px-2 py-1.5 text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text)] border border-dashed border-[var(--color-border)] rounded hover:border-[var(--color-border-hover)] transition-colors cursor-pointer shrink-0"
            >
              <Plus className="size-3" />
              Add agent
            </button>
          </div>

          {/* Active subtab form */}
          <AnimatePresence mode="wait">
            <motion.div
              key={activeSubtab}
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.12 }}
              className="flex flex-col gap-3"
            >
              {subtabErrors[activeSubtab] && (
                <p className="text-xs text-red-400">{subtabErrors[activeSubtab]}</p>
              )}
              <AgentFieldsForm
                values={activeAgent}
                onChange={(next) => patchAgent(activeSubtab, next)}
                active={active}
                hideNamespaceOnly
                idPrefix={`squad-agent-${activeSubtab}`}
              />
            </motion.div>
          </AnimatePresence>
        </div>

        {error && <p className="text-sm text-red-400">{error}</p>}
      </div>

      <div className="mt-4 shrink-0 flex justify-end gap-2">
        <Button variant="secondary" type="button" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={submitting}>
          {submitting ? "Creating..." : `Create Squad (${agents.length} agent${agents.length !== 1 ? "s" : ""})`}
        </Button>
      </div>
    </form>
  );
}
