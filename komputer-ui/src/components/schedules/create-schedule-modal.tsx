"use client";

import { useState, useRef, useEffect } from "react";
import { useRouter } from "next/navigation";
import { AnimatePresence, motion } from "framer-motion";
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
import { ChevronRight, Check } from "lucide-react";
import { NamespaceSelector } from "@/components/shared/namespace-selector";
import { createSchedule, listAgents } from "@/lib/api";
import type { CreateScheduleRequest } from "@/lib/types";
import { LIFECYCLES } from "@/lib/constants";

type CreateScheduleModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: () => void;
};

const NAME_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/;

export function CreateScheduleModal({ open, onOpenChange, onCreated }: CreateScheduleModalProps) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [namespace, setNamespace] = useState("default");
  const [cron, setCron] = useState("");
  const [instructions, setInstructions] = useState("");
  const [timezone, setTimezone] = useState("UTC");
  const [autoDelete, setAutoDelete] = useState(false);
  const [keepAgents, setKeepAgents] = useState(false);
  const [agentRef, setAgentRef] = useState("");
  const [lifecycle, setLifecycle] = useState("Sleep");
  const [availableAgents, setAvailableAgents] = useState<{ name: string; namespace: string }[]>([]);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    listAgents(namespace || undefined)
      .then((res) => setAvailableAgents((res.agents ?? []).map((a) => ({ name: a.name, namespace: a.namespace }))))
      .catch(() => setAvailableAgents([]));
  }, [open, namespace]);

  function resetForm() {
    setName("");
    setNamespace("default");
    setCron("");
    setInstructions("");
    setTimezone("UTC");
    setAutoDelete(false);
    setKeepAgents(false);
    setAgentRef("");
    setLifecycle("Sleep");
    setAdvancedOpen(false);
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
        namespace: namespace.trim() || undefined,
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
      <DialogContent>
        <form onSubmit={handleSubmit} className="flex flex-col min-h-0 flex-1">
          <DialogHeader>
            <DialogTitle>Create Schedule</DialogTitle>
            <DialogDescription>
              Schedule recurring agent tasks on a cron expression.
            </DialogDescription>
          </DialogHeader>

          <div ref={scrollRef} className="mt-4 flex flex-col gap-4 overflow-y-auto flex-1 pr-1">
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

            <NamespaceSelector value={namespace} onChange={setNamespace} />

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="schedule-cron">Cron Expression</Label>
              <Input
                id="schedule-cron"
                placeholder="0 9 * * MON-FRI"
                value={cron}
                onChange={(e) => setCron(e.target.value)}
                autoComplete="off"
              />
              <p className="text-[10px] text-[var(--color-text-muted)]">
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
                style={{ minHeight: 200 }}
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

            {/* Advanced section */}
            <div className="rounded-md border border-[var(--color-border)]">
              <button
                type="button"
                onClick={() => setAdvancedOpen(!advancedOpen)}
                className="flex w-full items-center gap-2 px-3 py-2 text-left cursor-pointer hover:bg-[var(--color-surface-hover)] transition-colors"
              >
                <ChevronRight
                  className={`size-3.5 shrink-0 text-[var(--color-text-secondary)] transition-transform duration-200 ${advancedOpen ? "rotate-90" : ""}`}
                />
                <span className="text-xs font-medium text-[var(--color-text-secondary)]">Advanced</span>
              </button>
              <AnimatePresence initial={false}>
                {advancedOpen && (
                  <motion.div
                    initial={{ height: 0, opacity: 0, overflow: "hidden" }}
                    animate={{ height: "auto", opacity: 1, overflow: "visible", transitionEnd: { overflow: "visible" } }}
                    exit={{ height: 0, opacity: 0, overflow: "hidden" }}
                    transition={{ duration: 0.2, ease: "easeOut" }}
                    onAnimationComplete={() => {
                      if (advancedOpen) scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" });
                    }}
                  >
                    <div className="border-t border-[var(--color-border)] px-3 py-3 flex flex-col gap-4">
                      {/* Cleanup toggles */}
                      <div className="flex flex-col gap-1.5">
                        <Label>Cleanup</Label>
                        <div className="flex flex-wrap gap-1.5">
                          <button
                            type="button"
                            onClick={() => {
                              const next = !autoDelete;
                              setAutoDelete(next);
                              if (!next) setKeepAgents(false);
                            }}
                            className={`text-xs px-2.5 py-1 rounded-full border transition-colors cursor-pointer ${
                              autoDelete
                                ? "border-[var(--color-text)] bg-white/10 text-[var(--color-text)]"
                                : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
                            }`}
                          >
                            {autoDelete && <Check className="inline size-2.5 mr-1" />}
                            Delete after first run
                          </button>
                          {autoDelete && (
                            <button
                              type="button"
                              onClick={() => setKeepAgents(!keepAgents)}
                              className={`text-xs px-2.5 py-1 rounded-full border transition-colors cursor-pointer ${
                                keepAgents
                                  ? "border-[var(--color-text)] bg-white/10 text-[var(--color-text)]"
                                  : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
                              }`}
                            >
                              {keepAgents && <Check className="inline size-2.5 mr-1" />}
                              Keep agents after deletion
                            </button>
                          )}
                        </div>
                      </div>

                      {/* Agent reference */}
                      <div className="flex flex-col gap-1.5">
                        <Label>Agent Reference</Label>
                        <Select value={agentRef || "none"} onValueChange={(v) => setAgentRef(v === "none" ? "" : v)}>
                          <SelectTrigger className="w-full">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="none">Create new agent</SelectItem>
                            {availableAgents.map((a) => (
                              <SelectItem key={a.name} value={a.name}>
                                {a.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>

                      {/* Lifecycle — only when no agent ref */}
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
                    </div>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>

            {error && (
              <p className="text-sm text-red-400">{error}</p>
            )}
          </div>

          <DialogFooter className="mt-4 shrink-0">
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
