"use client";

import { useState, useEffect } from "react";
import { Users } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/kit/dialog";
import { Button } from "@/components/kit/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/kit/select";
import { Input } from "@/components/kit/input";
import { Label } from "@/components/kit/label";
import { listAgents, createSquad, addSquadMember, patchAgent } from "@/lib/api";
import { Moon } from "lucide-react";
import type { Squad, AgentResponse } from "@/lib/types";

type TeamUpDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agentName: string;
  agentNamespace: string;
  agentStatus?: string;
  squads: Squad[];
  onSuccess?: () => void;
};

export function TeamUpDialog({
  open,
  onOpenChange,
  agentName,
  agentNamespace,
  agentStatus,
  squads,
  onSuccess,
}: TeamUpDialogProps) {
  const [agents, setAgents] = useState<AgentResponse[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<string>("");
  const [squadName, setSquadName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [agentSearch, setAgentSearch] = useState("");

  const filteredAgents = agents.filter((a) => {
    const q = agentSearch.trim().toLowerCase();
    if (!q) return true;
    return a.name.toLowerCase().includes(q) || a.namespace.toLowerCase().includes(q);
  });

  // Load all agents when dialog opens
  useEffect(() => {
    if (!open) return;
    listAgents(agentNamespace)
      .then((res) => {
        setAgents((res.agents || []).filter((a) => a.name !== agentName));
      })
      .catch(() => {});
  }, [open, agentName, agentNamespace]);

  // Auto-generate squad name when other agent is selected
  useEffect(() => {
    if (selectedAgent && !squadName) {
      setSquadName(`${agentName}-${selectedAgent}-squad`);
    }
  }, [selectedAgent, agentName, squadName]);

  // Find which squad (if any) the selected agent belongs to
  const selectedAgentSquad = selectedAgent
    ? squads.find((s) =>
        s.members.some((m) => m.name === selectedAgent)
      )
    : null;

  // Both agents must be asleep before teaming up — adopting a running agent
  // would leave its solo pod stranded next to the new squad pod.
  const currentAgentNotAsleep = !!agentStatus && agentStatus !== "Sleeping";
  const partnerAgent = agents.find((a) => a.name === selectedAgent) ?? null;
  const partnerNotAsleep =
    !!partnerAgent && !selectedAgentSquad && partnerAgent.status !== "Sleeping";

  // Build the list of agents the user needs to sleep before they can team up.
  // Each entry carries its namespace so the inline Sleep button can call patch.
  const notAsleepAgents: { name: string; namespace: string; status?: string }[] = [];
  if (currentAgentNotAsleep) {
    notAsleepAgents.push({ name: agentName, namespace: agentNamespace, status: agentStatus });
  }
  if (partnerNotAsleep && partnerAgent) {
    notAsleepAgents.push({ name: partnerAgent.name, namespace: partnerAgent.namespace, status: partnerAgent.status });
  }
  const blockingAsleep = notAsleepAgents.length > 0;

  const [sleepingAgents, setSleepingAgents] = useState<Record<string, boolean>>({});
  async function handleSleepAgent(name: string, namespace: string) {
    const key = `${namespace}/${name}`;
    setSleepingAgents((s) => ({ ...s, [key]: true }));
    setError(null);
    try {
      await patchAgent(name, { lifecycle: "Sleep" }, namespace);
      // Refresh the agent list so the warning recomputes from fresh status.
      const res = await listAgents(agentNamespace);
      setAgents((res.agents || []).filter((a) => a.name !== agentName));
      // Also bump agentStatus locally if this was the current agent — the parent
      // owns it via prop, but we can hint by removing the entry from the list.
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : `Failed to sleep ${name}`);
    } finally {
      setSleepingAgents((s) => ({ ...s, [key]: false }));
    }
  }

  async function handleSubmit() {
    if (!selectedAgent) return;
    setLoading(true);
    setError(null);
    try {
      if (selectedAgentSquad) {
        // Selected agent is in a squad — join that squad
        await addSquadMember(selectedAgentSquad.name, selectedAgentSquad.namespace, {
          ref: { name: agentName, namespace: agentNamespace },
        });
      } else {
        // Neither is in a squad — create a new one with both
        const name = squadName.trim() || `${agentName}-${selectedAgent}-squad`;
        await createSquad({
          name,
          namespace: agentNamespace,
          members: [
            { ref: { name: agentName, namespace: agentNamespace } },
            { ref: { name: selectedAgent, namespace: agentNamespace } },
          ],
        });
      }
      onSuccess?.();
      onOpenChange(false);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to team up");
    } finally {
      setLoading(false);
    }
  }

  const handleClose = () => {
    setSelectedAgent("");
    setSquadName("");
    setError(null);
    setAgentSearch("");
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-md mx-auto bg-[var(--color-surface)] border-[var(--color-border)] text-[var(--color-text)]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-[var(--color-text)]">
            <Users className="size-4 text-[var(--color-brand-blue)]" />
            Team Up
          </DialogTitle>
          <DialogDescription className="text-[var(--color-text-secondary)]">
            Pick an agent to team up with. If that agent is already in a squad,{" "}
            <span className="font-medium text-[var(--color-text)]">{agentName}</span> will join it.
            Otherwise a new squad is created with both agents.
          </DialogDescription>
        </DialogHeader>

        <div className="mt-4 space-y-4">
          <div className="flex flex-col gap-1.5">
            <Label>Agent to team up with</Label>
            <Select value={selectedAgent} onValueChange={setSelectedAgent}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Select an agent..." />
              </SelectTrigger>
              <SelectContent>
                <div
                  className="sticky top-0 z-10 -mt-1 mb-1 bg-[var(--color-surface)] px-2 py-1.5 border-b border-[var(--color-border)]"
                  onPointerDown={(e) => e.stopPropagation()}
                  onKeyDown={(e) => e.stopPropagation()}
                >
                  <Input
                    autoFocus
                    type="text"
                    placeholder="Search agents..."
                    value={agentSearch}
                    onChange={(e) => setAgentSearch(e.target.value)}
                    className="h-7 text-sm"
                  />
                </div>
                {filteredAgents.map((a) => (
                  <SelectItem key={a.name} value={a.name}>
                    {a.name}
                    {squads.some((s) => s.members.some((m) => m.name === a.name)) && (
                      <span className="ml-2 text-[10px] text-[var(--color-brand-blue)]">(in squad)</span>
                    )}
                  </SelectItem>
                ))}
                {filteredAgents.length === 0 && (
                  <p className="px-3 py-2 text-xs text-[var(--color-text-muted)]">
                    {agents.length === 0 ? "No other agents available" : "No agents match your search"}
                  </p>
                )}
              </SelectContent>
            </Select>
          </div>

          {selectedAgent && !selectedAgentSquad && (
            <div className="flex flex-col gap-1.5">
              <Label>Squad name</Label>
              <Input
                value={squadName}
                onChange={(e) => setSquadName(e.target.value)}
                placeholder={`${agentName}-${selectedAgent}-squad`}
              />
            </div>
          )}

          {selectedAgentSquad && (
            <p className="text-xs text-[var(--color-text-secondary)] rounded-md border border-[var(--color-border)] bg-[var(--color-surface-hover)] px-3 py-2">
              <span className="font-medium text-[var(--color-text)]">{selectedAgent}</span> is in squad{" "}
              <span className="font-medium text-[var(--color-brand-blue)]">{selectedAgentSquad.name}</span>.{" "}
              {agentName} will be added to that squad.
            </p>
          )}

          {blockingAsleep && (
            <div className="rounded-[var(--radius-md)] border border-amber-500/40 bg-amber-500/5 p-3 text-sm text-amber-300/90 space-y-2">
              <p>
                {notAsleepAgents.length === 1 ? "This agent" : "These agents"} must be asleep before teaming up:
              </p>
              <ul className="space-y-1.5">
                {notAsleepAgents.map((a) => {
                  const key = `${a.namespace}/${a.name}`;
                  const busy = !!sleepingAgents[key];
                  return (
                    <li key={key} className="flex items-center justify-between gap-2 rounded px-2 py-1">
                      <span className="text-sm">
                        <span className="font-medium text-amber-200">{a.name}</span>
                        <span className="ml-1.5 text-xs text-amber-300/70">({a.status || "unknown"})</span>
                      </span>
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => handleSleepAgent(a.name, a.namespace)}
                        disabled={busy}
                      >
                        <Moon className={`size-3 ${busy ? "animate-pulse" : ""}`} data-icon="inline-start" />
                        {busy ? "Sleeping…" : "Sleep"}
                      </Button>
                    </li>
                  );
                })}
              </ul>
              <p className="text-xs text-amber-300/70">
                The current agent&apos;s status updates after the next refresh.
              </p>
            </div>
          )}

          {error && (
            <p className="text-sm text-red-400">{error}</p>
          )}
        </div>

        <DialogFooter className="mt-4">
          <Button variant="secondary" onClick={handleClose} disabled={loading}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!selectedAgent || loading || blockingAsleep}
          >
            {loading ? "Teaming up..." : "Team Up"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
