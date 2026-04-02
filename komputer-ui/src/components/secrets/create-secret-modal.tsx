"use client";

import { useState, useCallback } from "react";
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
import { Label } from "@/components/kit/label";
import { Plus, Trash2 } from "lucide-react";
import { NamespaceSelector } from "@/components/shared/namespace-selector";
import { createSecretResource } from "@/lib/api";

type KeyValueEntry = { key: string; value: string };

const NAME_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/;

type CreateSecretModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: () => void;
};

export function CreateSecretModal({ open, onOpenChange, onCreated }: CreateSecretModalProps) {
  const [name, setName] = useState("");
  const [namespace, setNamespace] = useState("default");
  const [pairs, setPairs] = useState<KeyValueEntry[]>([{ key: "", value: "" }]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function resetForm() {
    setName("");
    setNamespace("default");
    setPairs([{ key: "", value: "" }]);
    setError(null);
  }

  function validate(): string | null {
    if (!name.trim()) return "Name is required.";
    if (!NAME_PATTERN.test(name)) return "Name must be lowercase letters, numbers, and hyphens only.";
    const filled = pairs.filter((p) => p.key.trim() && p.value.trim());
    if (filled.length === 0) return "At least one key-value pair is required.";
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
      const data: Record<string, string> = {};
      for (const p of pairs) {
        const k = p.key.trim();
        const v = p.value.trim();
        if (k && v) data[k] = v;
      }

      await createSecretResource({
        name: name.trim(),
        data,
        namespace: namespace.trim() || undefined,
      });

      resetForm();
      onOpenChange(false);
      onCreated?.();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to create secret.");
    } finally {
      setSubmitting(false);
    }
  }

  const addPair = useCallback(() => {
    setPairs((prev) => [...prev, { key: "", value: "" }]);
  }, []);

  const removePair = useCallback((index: number) => {
    setPairs((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const updatePair = useCallback((index: number, field: "key" | "value", val: string) => {
    setPairs((prev) => prev.map((p, i) => (i === index ? { ...p, [field]: val } : p)));
  }, []);

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
            <DialogTitle>Create Secret</DialogTitle>
            <DialogDescription>
              Store sensitive key-value pairs as a Kubernetes Secret.
            </DialogDescription>
          </DialogHeader>

          <div className="mt-4 flex flex-col gap-4 overflow-y-auto flex-1 pr-1">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="secret-name">Name</Label>
              <Input
                id="secret-name"
                placeholder="github-token"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoComplete="off"
              />
            </div>

            <NamespaceSelector value={namespace} onChange={setNamespace} />

            <div className="flex flex-col gap-1.5">
              <Label>Key-Value Pairs</Label>
              <div className="flex flex-col gap-2">
                {pairs.map((pair, index) => (
                  <div key={index} className="flex items-center gap-2">
                    <Input
                      placeholder="KEY"
                      value={pair.key}
                      onChange={(e) => updatePair(index, "key", e.target.value)}
                      autoComplete="off"
                      className="flex-1"
                    />
                    <Input
                      type="password"
                      placeholder="value"
                      value={pair.value}
                      onChange={(e) => updatePair(index, "value", e.target.value)}
                      autoComplete="off"
                      className="flex-1"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => removePair(index)}
                      className="shrink-0"
                      disabled={pairs.length === 1}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={addPair}
                  className="w-fit"
                >
                  <Plus className="mr-1 h-4 w-4" />
                  Add Key
                </Button>
              </div>
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
              {submitting ? "Creating..." : "Create Secret"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
