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
import { createSkill } from "@/lib/api";

const NAME_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/;

type CreateSkillModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: () => void;
};

export function CreateSkillModal({ open, onOpenChange, onCreated }: CreateSkillModalProps) {

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
    if (!description.trim()) return "Description is required.";
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
      await createSkill({
        name: name.trim(),
        description: description.trim(),
        content: content.trim(),
        namespace: namespace.trim() || undefined,
      });
      resetForm();
      onOpenChange(false);
      onCreated?.();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to create skill.");
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
            <DialogTitle>Create Skill</DialogTitle>
            <DialogDescription>
              Create a reusable skill that can be attached to agents.
            </DialogDescription>
          </DialogHeader>

          <div className="mt-4 flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="skill-name">Name</Label>
              <Input
                id="skill-name"
                placeholder="python-expert"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoComplete="off"
              />
            </div>

            <NamespaceSelector value={namespace} onChange={setNamespace} />

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="skill-description">Description</Label>
              <Input
                id="skill-description"
                placeholder="Short description of what this skill does"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                autoComplete="off"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="skill-content">Content</Label>
              <Textarea
                id="skill-content"
                placeholder="Write the skill instructions, examples, or knowledge..."
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
              {submitting ? "Creating..." : "Create Skill"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
