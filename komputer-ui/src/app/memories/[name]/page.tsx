"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { motion } from "framer-motion";
import { Trash2, Brain, Users, Save, Check, Pencil, Eye } from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { Button } from "@/components/kit/button";
import { Badge } from "@/components/kit/badge";
import { Textarea } from "@/components/kit/textarea";
import { Input } from "@/components/kit/input";
import { Label } from "@/components/kit/label";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { RelativeTime } from "@/components/shared/relative-time";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { getMemory, deleteMemory, patchMemory } from "@/lib/api";
import type { MemoryResponse } from "@/lib/types";

export default function MemoryDetailPage() {
  const params = useParams<{ name: string }>();
  const searchParams = useSearchParams();
  const router = useRouter();
  const memoryName = params.name;
  const memoryNs = searchParams.get("namespace") || undefined;

  const [memory, setMemory] = useState<MemoryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const showLoading = useDelayedLoading(loading);
  const [error, setError] = useState<string | null>(null);

  const fetchMemory = useCallback(async () => {
    if (!memoryName) return;
    try {
      const data = await getMemory(memoryName, memoryNs);
      setMemory(data);
      setError(null);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [memoryName, memoryNs]);

  useEffect(() => {
    fetchMemory();
  }, [fetchMemory]);

  const handleDelete = async () => {
    if (!memoryName) return;
    try {
      await deleteMemory(memoryName, memoryNs);
      router.push("/memories");
    } catch {}
  };

  if (showLoading) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex flex-1 items-center justify-center text-sm text-[var(--color-text-secondary)]">
          Loading memory...
        </div>
      </div>
    );
  }

  if (loading) return null;

  if (error || !memory) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex flex-1 items-center justify-center">
          <div className="rounded-lg border border-red-400/20 bg-red-400/5 p-6 text-center">
            <p className="text-sm text-red-400">{error ?? "Memory not found"}</p>
            <Button variant="ghost" size="sm" className="mt-3" onClick={() => router.push("/memories")}>
              Back to memories
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, ease: "easeOut" }}
      className="flex h-full flex-col"
    >
      {/* Info bar */}
      <div className="flex flex-wrap items-center gap-3 border-b border-[var(--color-border)] bg-[var(--color-bg)] px-6 py-3">
        <Brain className="size-4 text-[var(--color-brand-violet)]" />
        <span className="text-sm text-[var(--color-text-muted)]">Namespace:</span>
        <Badge variant="outline" className="text-xs font-mono">{memory.namespace}</Badge>
        {memory.description && (
          <>
            <span className="text-sm text-[var(--color-text-muted)]">Description:</span>
            <span className="text-sm text-[var(--color-text-secondary)]">{memory.description}</span>
          </>
        )}
        <span className="text-sm text-[var(--color-text-muted)]">Agents:</span>
        <span className="text-sm text-[var(--color-text-secondary)] flex items-center gap-1">
          <Users className="size-3" /> {memory.attachedAgents}
        </span>

        <div className="ml-auto flex items-center gap-3">
          <RelativeTime timestamp={memory.createdAt} />
          <ConfirmDialog
            title="Delete Memory"
            description={`Are you sure you want to delete "${memory.name}"? This action cannot be undone.`}
            onConfirm={handleDelete}
            trigger={
              <Button variant="destructive" size="sm">
                <Trash2 className="size-3" data-icon="inline-start" />
                Delete
              </Button>
            }
          />
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        <div className="mx-auto max-w-4xl space-y-6">
          {/* Editable memory */}
          <MemoryEditor memory={memory} namespace={memoryNs} onSaved={fetchMemory} />

          {/* Attached agents */}
          {memory.agentNames && memory.agentNames.length > 0 && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, delay: 0.2 }}
              className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-5 space-y-3"
            >
              <h3 className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)]">Attached Agents</h3>
              <div className="flex flex-wrap gap-2">
                {memory.agentNames.map((name) => (
                  <Link key={name} href={`/agents/${name}?namespace=${memory.namespace}`}>
                    <Badge variant="secondary" className="cursor-pointer text-[var(--color-brand-blue-light)] border-[var(--color-brand-blue)]/30 hover:bg-[var(--color-brand-blue)]/10 hover:border-[var(--color-brand-blue)]/50 transition-colors">
                      {name}
                    </Badge>
                  </Link>
                ))}
              </div>
            </motion.div>
          )}
        </div>
      </div>
    </motion.div>
  );
}

function MemoryEditor({ memory, namespace, onSaved }: { memory: MemoryResponse; namespace?: string; onSaved: () => void }) {
  const [content, setContent] = useState(memory.content);
  const [description, setDescription] = useState(memory.description ?? "");
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [mode, setMode] = useState<"preview" | "edit">("preview");

  const hasChanges = content !== memory.content || description !== (memory.description ?? "");

  async function handleSave() {
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      const patch: Record<string, string> = {};
      if (content !== memory.content) patch.content = content;
      if (description !== (memory.description ?? "")) patch.description = description;
      await patchMemory(memory.name, patch, namespace);
      setSaved(true);
      onSaved();
      setTimeout(() => setSaved(false), 2000);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save");
    } finally {
      setSaving(false);
    }
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, delay: 0.1 }}
      className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-5 space-y-4"
    >
      <div className="flex items-center justify-between">
        <h3 className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)]">Content</h3>
        <Button
          size="sm"
          variant="ghost"
          onClick={() => setMode((m) => (m === "preview" ? "edit" : "preview"))}
        >
          {mode === "preview" ? (
            <><Pencil className="size-3 mr-1" /> Edit</>
          ) : (
            <><Eye className="size-3 mr-1" /> Preview</>
          )}
        </Button>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label>Description</Label>
        {mode === "edit" ? (
          <Input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Short description (optional)"
          />
        ) : (
          <p className="text-sm text-[var(--color-text-secondary)] min-h-[2rem]">
            {description || <span className="italic text-[var(--color-text-muted)]">No description</span>}
          </p>
        )}
      </div>

      <div className="flex flex-col gap-1.5">
        <Label>Memory Content</Label>
        {mode === "edit" ? (
          <Textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            style={{ minHeight: 300 }}
          />
        ) : (
          <div className="prose-chat text-sm text-[var(--color-text)] rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-bg)] p-4 min-h-[300px]">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
          </div>
        )}
      </div>

      {error && <p className="text-sm text-red-400">{error}</p>}

      {mode === "edit" && (
        <div className="flex items-center gap-2">
          <Button size="sm" onClick={handleSave} disabled={!hasChanges || saving}>
            {saved ? (
              <><Check className="size-3 mr-1" /> Saved</>
            ) : saving ? (
              "Saving..."
            ) : (
              <><Save className="size-3 mr-1" /> Save Changes</>
            )}
          </Button>
        </div>
      )}
    </motion.div>
  );
}
