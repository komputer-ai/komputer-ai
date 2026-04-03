"use client";

import { useState } from "react";
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
import { NamespaceSelector } from "@/components/shared/namespace-selector";
import { createMemory } from "@/lib/api";

const NAME_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/;

type CreateMemoryModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: () => void;
};

export function CreateMemoryModal({ open, onOpenChange, onCreated }: CreateMemoryModalProps) {
  const [name, setName] = useState("");
  const [namespace, setNamespace] = useState("default");
  const [content, setContent] = useState("");
  const [description, setDescription] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function resetForm() {
    setName("");
    setNamespace("default");
    setContent("");
    setDescription("");
    setError(null);
  }

  function validate(): string | null {
    if (!name.trim()) return "Name is required.";
    if (!NAME_PATTERN.test(name)) return "Name must be lowercase letters, numbers, and hyphens only.";
    if (!content.trim()) return "Content is required.";
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
      await createMemory({
        name: name.trim(),
        content: content.trim(),
        description: description.trim() || undefined,
        namespace: namespace.trim() || undefined,
      });
      resetForm();
      onOpenChange(false);
      onCreated?.();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to create memory.");
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
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Memory</DialogTitle>
            <DialogDescription>
              Create a persistent knowledge note that can be attached to agents.
            </DialogDescription>
          </DialogHeader>

          <div className="mt-4 flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="memory-name">Name</Label>
              <Input
                id="memory-name"
                placeholder="k8s-debugging-guide"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoComplete="off"
              />
            </div>

            <NamespaceSelector value={namespace} onChange={setNamespace} />

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="memory-description">Description</Label>
              <Input
                id="memory-description"
                placeholder="Short description (optional)"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                autoComplete="off"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="memory-content">Content</Label>
              <Textarea
                id="memory-content"
                placeholder="Write the knowledge, context, or instructions..."
                value={content}
                onChange={(e) => setContent(e.target.value)}
                style={{ minHeight: 200 }}
              />
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
              {submitting ? "Creating..." : "Create Memory"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
