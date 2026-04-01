"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { motion } from "framer-motion";
import { Trash2, Wand2, Users, Save, Check } from "lucide-react";

import { Button } from "@/components/kit/button";
import { Badge } from "@/components/kit/badge";
import { Textarea } from "@/components/kit/textarea";
import { Input } from "@/components/kit/input";
import { Label } from "@/components/kit/label";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { RelativeTime } from "@/components/shared/relative-time";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { getSkill, deleteSkill, patchSkill } from "@/lib/api";
import type { SkillResponse } from "@/lib/types";

export default function SkillDetailPage() {
  const params = useParams<{ name: string }>();
  const searchParams = useSearchParams();
  const router = useRouter();
  const skillName = params.name;
  const skillNs = searchParams.get("namespace") || undefined;

  const [skill, setSkill] = useState<SkillResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const showLoading = useDelayedLoading(loading);
  const [error, setError] = useState<string | null>(null);

  const fetchSkill = useCallback(async () => {
    if (!skillName) return;
    try {
      const data = await getSkill(skillName, skillNs);
      setSkill(data);
      setError(null);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [skillName, skillNs]);

  useEffect(() => {
    fetchSkill();
  }, [fetchSkill]);

  const handleDelete = async () => {
    if (!skillName) return;
    try {
      await deleteSkill(skillName, skillNs);
      router.push("/skills");
    } catch {}
  };

  if (showLoading) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex flex-1 items-center justify-center text-sm text-[var(--color-text-secondary)]">
          Loading skill...
        </div>
      </div>
    );
  }

  if (loading) return null;

  if (error || !skill) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex flex-1 items-center justify-center">
          <div className="rounded-lg border border-red-400/20 bg-red-400/5 p-6 text-center">
            <p className="text-sm text-red-400">{error ?? "Skill not found"}</p>
            <Button variant="ghost" size="sm" className="mt-3" onClick={() => router.push("/skills")}>
              Back to skills
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
        <Wand2 className="size-4 text-violet-400" />
        <span className="text-sm text-[var(--color-text-muted)]">Namespace:</span>
        <Badge variant="outline" className="text-xs font-mono">{skill.namespace}</Badge>
        {skill.description && (
          <>
            <span className="text-sm text-[var(--color-text-muted)]">Description:</span>
            <span className="text-sm text-[var(--color-text-secondary)]">{skill.description}</span>
          </>
        )}
        <span className="text-sm text-[var(--color-text-muted)]">Agents:</span>
        <span className="text-sm text-[var(--color-text-secondary)] flex items-center gap-1">
          <Users className="size-3" /> {skill.attachedAgents}
        </span>

        <div className="ml-auto flex items-center gap-3">
          <RelativeTime timestamp={skill.createdAt} />
          <ConfirmDialog
            title="Delete Skill"
            description={`Are you sure you want to delete "${skill.name}"? This action cannot be undone.`}
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
          {/* Editable skill */}
          <SkillEditor skill={skill} namespace={skillNs} onSaved={fetchSkill} />

          {/* Attached agents */}
          {skill.agentNames && skill.agentNames.length > 0 && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, delay: 0.2 }}
              className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-5 space-y-3"
            >
              <h3 className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)]">Attached Agents</h3>
              <div className="flex flex-wrap gap-2">
                {skill.agentNames.map((name) => (
                  <Link key={name} href={`/agents/${name}`}>
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

function SkillEditor({ skill, namespace, onSaved }: { skill: SkillResponse; namespace?: string; onSaved: () => void }) {
  const [content, setContent] = useState(skill.content);
  const [description, setDescription] = useState(skill.description ?? "");
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const hasChanges = content !== skill.content || description !== (skill.description ?? "");

  async function handleSave() {
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      const patch: Record<string, string> = {};
      if (content !== skill.content) patch.content = content;
      if (description !== (skill.description ?? "")) patch.description = description;
      await patchSkill(skill.name, patch, namespace);
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
      <h3 className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)]">Content</h3>

      <div className="flex flex-col gap-1.5">
        <Label>Description</Label>
        <Input
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Short description of what this skill does"
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <Label>Skill Content</Label>
        <Textarea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          style={{ minHeight: 300 }}
        />
      </div>

      {error && <p className="text-sm text-red-400">{error}</p>}

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
    </motion.div>
  );
}
