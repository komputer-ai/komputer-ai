"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/kit/dialog";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/kit/tabs";
import { createAgent } from "@/lib/api";
import type { CreateAgentRequest } from "@/lib/types";
import type { AgentTemplate } from "@/lib/create-agent-modal-context";
import {
  type AgentFormValues,
  makeDefaultAgentFormValues,
  buildCreateAgentRequest,
} from "./agent-fields-form";
import { SoloModeForm } from "./solo-mode-form";
import { SquadModeForm, type SquadState } from "./squad-mode-form";
import { TeamUpModeForm, type TeamUpState } from "./team-up-mode-form";

type CreateAgentModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: () => void;
  initialValues?: AgentTemplate | null;
};

type Mode = "solo" | "squad" | "team-up";

const NAME_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/;

function defaultSquadState(): SquadState {
  return {
    squadName: "",
    namespace: "default",
    activeSubtab: 0,
    agents: [makeDefaultAgentFormValues()],
  };
}

function defaultTeamUpState(): TeamUpState {
  return {
    values: makeDefaultAgentFormValues(),
    teamUpWithAgent: "",
    squadName: "",
  };
}

export function CreateAgentModal({ open, onOpenChange, onCreated, initialValues }: CreateAgentModalProps) {
  const router = useRouter();
  const [mode, setMode] = useState<Mode>("solo");

  // Per-mode state — preserved across tab switches
  const [soloValues, setSoloValues] = useState<AgentFormValues>(makeDefaultAgentFormValues());
  const [squadState, setSquadState] = useState<SquadState>(defaultSquadState());
  const [teamUpState, setTeamUpState] = useState<TeamUpState>(defaultTeamUpState());

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function resetAll() {
    setMode("solo");
    setSoloValues(makeDefaultAgentFormValues());
    setSquadState(defaultSquadState());
    setTeamUpState(defaultTeamUpState());
    setSubmitting(false);
    setError(null);
  }

  // Apply initialValues when modal opens (only seeds Solo mode)
  useEffect(() => {
    if (open && initialValues) {
      setSoloValues(makeDefaultAgentFormValues({
        name: initialValues.name,
        instructions: initialValues.instructions,
        model: initialValues.model,
        lifecycle: initialValues.lifecycle,
        role: initialValues.role,
        templateRef: initialValues.templateRef ?? "default",
      }));
      setError(null);
    }
  }, [open, initialValues]);

  function handleOpenChange(nextOpen: boolean) {
    onOpenChange(nextOpen);
    if (!nextOpen) resetAll();
  }

  function handleModeChange(nextMode: string) {
    setMode(nextMode as Mode);
    setError(null);
  }

  // ---- Solo submit ----
  async function handleSoloSubmit() {
    if (!soloValues.name.trim()) { setError("Name is required."); return; }
    if (!NAME_PATTERN.test(soloValues.name)) { setError("Name must be lowercase letters, numbers, and hyphens only."); return; }
    if (!soloValues.instructions.trim()) { setError("Instructions are required."); return; }

    setSubmitting(true);
    setError(null);

    try {
      const req = buildCreateAgentRequest(soloValues) as CreateAgentRequest;
      await createAgent(req);
      const agentName = soloValues.name.trim();
      const agentNs = soloValues.namespace.trim() || undefined;
      const agentInstructions = soloValues.instructions.trim();
      resetAll();
      onOpenChange(false);
      onCreated?.();
      const params = new URLSearchParams();
      if (agentNs) params.set("namespace", agentNs);
      params.set("pending", agentInstructions);
      router.push(`/agents/${agentName}?${params.toString()}`);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to create agent.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-4xl h-[85vh] flex flex-col overflow-hidden">
        <DialogHeader className="shrink-0">
          <DialogTitle>Create Agent</DialogTitle>
          <DialogDescription>
            Deploy a new Claude agent to your cluster.
          </DialogDescription>
        </DialogHeader>

        <Tabs defaultValue="solo" value={mode} onValueChange={handleModeChange} className="mt-2 flex flex-col flex-1 min-h-0">
          <TabsList className="mb-4 shrink-0">
            <TabsTrigger value="solo">Solo</TabsTrigger>
            <TabsTrigger value="squad">Squad</TabsTrigger>
            <TabsTrigger value="team-up">Team Up</TabsTrigger>
          </TabsList>

          <TabsContent value="solo" className="flex-1 min-h-0 flex flex-col">
            <SoloModeForm
              values={soloValues}
              onChange={setSoloValues}
              active={mode === "solo"}
              submitting={submitting}
              error={mode === "solo" ? error : null}
              onSubmit={handleSoloSubmit}
              onCancel={() => handleOpenChange(false)}
            />
          </TabsContent>

          <TabsContent value="squad" className="flex-1 min-h-0 flex flex-col">
            <SquadModeForm
              state={squadState}
              onChange={setSquadState}
              active={mode === "squad"}
              error={mode === "squad" ? error : null}
              onError={(e) => setError(e)}
              onCreated={() => {
                resetAll();
                onCreated?.();
                onOpenChange(false);
              }}
              onCancel={() => handleOpenChange(false)}
            />
          </TabsContent>

          <TabsContent value="team-up" className="flex-1 min-h-0 flex flex-col">
            <TeamUpModeForm
              state={teamUpState}
              onChange={setTeamUpState}
              active={mode === "team-up"}
              error={mode === "team-up" ? error : null}
              onError={(e) => setError(e)}
              onCreated={() => {
                resetAll();
                onCreated?.();
                onOpenChange(false);
              }}
              onCancel={() => handleOpenChange(false)}
            />
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}
