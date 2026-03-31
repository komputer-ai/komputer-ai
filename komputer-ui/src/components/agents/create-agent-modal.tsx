"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/kit/dialog";
import { Button } from "@/components/kit/button";
import { Input } from "@/components/kit/input";
import { Textarea } from "@/components/kit/textarea";
import { Label } from "@/components/kit/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/kit/select";
import { Plus, Trash2 } from "lucide-react";
import { createAgent } from "@/lib/api";
import type { CreateAgentRequest } from "@/lib/types";
import type { AgentTemplate } from "@/lib/create-agent-modal-context";

type SecretEntry = { key: string; value: string };

type CreateAgentModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: () => void;
  initialValues?: AgentTemplate | null;
};

const NAME_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/;

const MODELS = [
  { value: "claude-sonnet-4-6", label: "claude-sonnet-4-6" },
  { value: "claude-opus-4-6", label: "claude-opus-4-6" },
  { value: "claude-haiku-4-5", label: "claude-haiku-4-5" },
];

const LIFECYCLES = [
  { value: "default", label: "Default (keep running)" },
  { value: "Sleep", label: "Sleep (preserve workspace)" },
  { value: "AutoDelete", label: "Auto Delete (one-shot)" },
];

export function CreateAgentModal({ open, onOpenChange, onCreated, initialValues }: CreateAgentModalProps) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [namespace, setNamespace] = useState("default");
  const [instructions, setInstructions] = useState("");
  const [model, setModel] = useState("claude-sonnet-4-6");
  const [lifecycle, setLifecycle] = useState("default");
  const [role, setRole] = useState<"manager" | "worker" | undefined>(undefined);
  const [secrets, setSecrets] = useState<SecretEntry[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function resetForm() {
    setName("");
    setNamespace("default");
    setInstructions("");
    setModel("claude-sonnet-4-6");
    setLifecycle("default");
    setRole(undefined);
    setSecrets([]);
    setError(null);
  }

  useEffect(() => {
    if (open && initialValues) {
      setName(initialValues.name);
      setInstructions(initialValues.instructions);
      setModel(initialValues.model);
      setLifecycle(initialValues.lifecycle);
      setRole(initialValues.role);
      if (initialValues.secrets) {
        setSecrets(
          Object.entries(initialValues.secrets).map(([key, value]) => ({ key, value }))
        );
      } else {
        setSecrets([]);
      }
      setError(null);
    }
  }, [open, initialValues]);

  function validate(): string | null {
    if (!name.trim()) return "Name is required.";
    if (!NAME_PATTERN.test(name))
      return "Name must be lowercase letters, numbers, and hyphens only.";
    if (!instructions.trim()) return "Instructions are required.";
    return null;
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const validationError = validate();
    if (validationError) {
      setError(validationError);
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      const secretsMap: Record<string, string> = {};
      for (const s of secrets) {
        const k = s.key.trim();
        const v = s.value.trim();
        if (k && v) secretsMap[k] = v;
      }

      const req: CreateAgentRequest = {
        name: name.trim(),
        instructions: instructions.trim(),
        model,
        namespace: namespace.trim() || undefined,
        lifecycle: lifecycle === "default" ? "" : (lifecycle as "" | "Sleep" | "AutoDelete"),
        role: role || undefined,
        secrets: Object.keys(secretsMap).length > 0 ? secretsMap : undefined,
      };
      await createAgent(req);
      const agentName = name.trim();
      const agentNs = namespace.trim() || undefined;
      const agentInstructions = instructions.trim();
      resetForm();
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

  const addSecret = useCallback(() => {
    setSecrets((prev) => [...prev, { key: "", value: "" }]);
  }, []);

  const removeSecret = useCallback((index: number) => {
    setSecrets((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const updateSecret = useCallback(
    (index: number, field: "key" | "value", val: string) => {
      setSecrets((prev) =>
        prev.map((s, i) => (i === index ? { ...s, [field]: val } : s))
      );
    },
    []
  );

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        onOpenChange(nextOpen);
        if (!nextOpen) resetForm();
      }}
    >
      <DialogContent>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Agent</DialogTitle>
            <DialogDescription>
              Deploy a new Claude agent to your cluster.
            </DialogDescription>
          </DialogHeader>

          <div className="mt-4 flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="agent-name">Name</Label>
              <Input
                id="agent-name"
                placeholder="my-agent"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoComplete="off"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="agent-namespace">Namespace</Label>
              <Input
                id="agent-namespace"
                placeholder="default"
                value={namespace}
                onChange={(e) => setNamespace(e.target.value)}
                autoComplete="off"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="agent-instructions">Instructions</Label>
              <Textarea
                id="agent-instructions"
                placeholder="Describe what this agent should do..."
                value={instructions}
                onChange={(e) => setInstructions(e.target.value)}
                style={{ minHeight: 200 }}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label>Model</Label>
              <Select value={model} onValueChange={(v) => v && setModel(v)}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {MODELS.map((m) => (
                    <SelectItem key={m.value} value={m.value}>
                      {m.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label>Lifecycle</Label>
              <Select value={lifecycle} onValueChange={(v) => v && setLifecycle(v)}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {LIFECYCLES.map((l) => (
                    <SelectItem key={l.value} value={l.value}>
                      {l.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label>Secrets</Label>
              <div className="flex flex-col gap-2">
                {secrets.map((secret, index) => (
                  <div key={index} className="flex items-center gap-2">
                    <Input
                      placeholder="KEY"
                      value={secret.key}
                      onChange={(e) => updateSecret(index, "key", e.target.value)}
                      autoComplete="off"
                      className="flex-1"
                    />
                    <Input
                      type="password"
                      placeholder="value"
                      value={secret.value}
                      onChange={(e) => updateSecret(index, "value", e.target.value)}
                      autoComplete="off"
                      className="flex-1"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => removeSecret(index)}
                      className="shrink-0"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={addSecret}
                  className="w-fit"
                >
                  <Plus className="mr-1 h-4 w-4" />
                  Add Secret
                </Button>
              </div>
            </div>

            {error && (
              <p className="text-sm text-red-400">{error}</p>
            )}
          </div>

          <DialogFooter className="mt-4">
            <Button variant="secondary" type="button" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? "Creating..." : "Create Agent"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
