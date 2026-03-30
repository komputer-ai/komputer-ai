"use client";

import { useState } from "react";
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
import { createSchedule } from "@/lib/api";
import type { CreateScheduleRequest } from "@/lib/types";

type CreateScheduleModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: () => void;
};

const NAME_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/;

const LIFECYCLES = [
  { value: "Sleep", label: "Sleep (preserve workspace)" },
  { value: "default", label: "Default (keep running)" },
  { value: "AutoDelete", label: "Auto Delete (one-shot)" },
];

export function CreateScheduleModal({ open, onOpenChange, onCreated }: CreateScheduleModalProps) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [cron, setCron] = useState("");
  const [instructions, setInstructions] = useState("");
  const [timezone, setTimezone] = useState("UTC");
  const [autoDelete, setAutoDelete] = useState(false);
  const [keepAgents, setKeepAgents] = useState(false);
  const [agentRef, setAgentRef] = useState("");
  const [lifecycle, setLifecycle] = useState("Sleep");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function resetForm() {
    setName("");
    setCron("");
    setInstructions("");
    setTimezone("UTC");
    setAutoDelete(false);
    setKeepAgents(false);
    setAgentRef("");
    setLifecycle("Sleep");
    setError(null);
  }

  function validate(): string | null {
    if (!name.trim()) return "Name is required.";
    if (!NAME_PATTERN.test(name))
      return "Name must be lowercase letters, numbers, and hyphens only.";
    if (!cron.trim()) return "Cron expression is required.";
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
      const req: CreateScheduleRequest = {
        name: name.trim(),
        schedule: cron.trim(),
        instructions: instructions.trim(),
        timezone: timezone.trim() || "UTC",
        autoDelete,
        keepAgents: autoDelete ? keepAgents : undefined,
      };

      if (agentRef.trim()) {
        req.agentName = agentRef.trim();
      } else {
        req.agent = {
          lifecycle: lifecycle === "default" ? "" : lifecycle,
        };
      }

      await createSchedule(req);
      const scheduleName = name.trim();
      resetForm();
      onOpenChange(false);
      onCreated?.();
      router.push(`/schedules/${scheduleName}`);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to create schedule.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        onOpenChange(nextOpen);
        if (!nextOpen) resetForm();
      }}
    >
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Schedule</DialogTitle>
            <DialogDescription>
              Schedule recurring agent tasks on a cron expression.
            </DialogDescription>
          </DialogHeader>

          <div className="mt-4 flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="schedule-name">Name</Label>
              <Input
                id="schedule-name"
                placeholder="daily-report"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoComplete="off"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="schedule-cron">Cron Expression</Label>
              <Input
                id="schedule-cron"
                placeholder="0 9 * * MON-FRI"
                value={cron}
                onChange={(e) => setCron(e.target.value)}
                autoComplete="off"
              />
              <p className="text-[10px] text-muted-foreground">
                0 9 * * MON-FRI = Weekdays 9am &nbsp;·&nbsp; */30 * * * * = Every 30min
              </p>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="schedule-instructions">Instructions</Label>
              <Textarea
                id="schedule-instructions"
                placeholder="Describe what the scheduled agent should do..."
                value={instructions}
                onChange={(e) => setInstructions(e.target.value)}
                className="min-h-24"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="schedule-tz">Timezone</Label>
              <Input
                id="schedule-tz"
                placeholder="e.g. Asia/Jerusalem"
                value={timezone}
                onChange={(e) => setTimezone(e.target.value)}
                autoComplete="off"
              />
            </div>

            <div className="flex items-center gap-2">
              <input
                id="schedule-autodelete"
                type="checkbox"
                checked={autoDelete}
                onChange={(e) => {
                  setAutoDelete(e.target.checked);
                  if (!e.target.checked) setKeepAgents(false);
                }}
                className="size-4 rounded accent-[var(--color-brand-blue)]"
              />
              <Label htmlFor="schedule-autodelete" className="text-sm font-normal">
                Delete after first run
              </Label>
            </div>

            {autoDelete && (
              <div className="flex items-center gap-2 pl-6">
                <input
                  id="schedule-keepagents"
                  type="checkbox"
                  checked={keepAgents}
                  onChange={(e) => setKeepAgents(e.target.checked)}
                  className="size-4 rounded accent-[var(--color-brand-blue)]"
                />
                <Label htmlFor="schedule-keepagents" className="text-sm font-normal">
                  Keep agents after schedule deletion
                </Label>
              </div>
            )}

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="schedule-agentref">Agent Reference (optional)</Label>
              <Input
                id="schedule-agentref"
                placeholder="Reference existing agent name"
                value={agentRef}
                onChange={(e) => setAgentRef(e.target.value)}
                autoComplete="off"
              />
            </div>

            {!agentRef.trim() && (
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
            )}

            {error && (
              <p className="text-sm text-red-400">{error}</p>
            )}
          </div>

          <DialogFooter className="mt-4">
            <Button variant="secondary" type="button" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? "Creating..." : "Create Schedule"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
