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
import { listAgents, createSquad, addSquadMember } from "@/lib/api";
import type { Squad, AgentResponse } from "@/lib/types";

type TeamUpDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agentName: string;
  agentNamespace: string;
  squads: Squad[];
  onSuccess?: () => void;
};

export function TeamUpDialog({
  open,
  onOpenChange,
  agentName,
  agentNamespace,
  squads,
  onSuccess,
}: TeamUpDialogProps) {
  const [agents, setAgents] = useState<AgentResponse[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<string>("");
  const [squadName, setSquadName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

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
                {agents.map((a) => (
                  <SelectItem key={a.name} value={a.name}>
                    {a.name}
                    {squads.some((s) => s.members.some((m) => m.name === a.name)) && (
                      <span className="ml-2 text-[10px] text-[var(--color-brand-blue)]">(in squad)</span>
                    )}
                  </SelectItem>
                ))}
                {agents.length === 0 && (
                  <SelectItem value="__none__">
                    No other agents available
                  </SelectItem>
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

          {error && (
            <p className="text-sm text-red-400">{error}</p>
          )}
        </div>

        <DialogFooter className="mt-4">
          <Button variant="secondary" onClick={handleClose} disabled={loading}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={!selectedAgent || loading}>
            {loading ? "Teaming up..." : "Team Up"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
