"use client";

import { useState, useMemo } from "react";
import { motion } from "framer-motion";

import { Plus } from "lucide-react";
import { Button } from "@/components/kit/button";
import { SkillCards } from "@/components/skills/skill-cards";
import { CreateSkillModal } from "@/components/skills/create-skill-modal";
import { EmptyState } from "@/components/shared/empty-state";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { ListFilterBar } from "@/components/shared/list-filter-bar";
import { useSkills } from "@/hooks/use-skills";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { usePageRefresh } from "@/components/layout/app-shell";
import { deleteSkill } from "@/lib/api";

export default function SkillsPage() {
  const { skills, loading, error, refresh } = useSkills();
  const showLoading = useDelayedLoading(loading);
  const [search, setSearch] = useState("");
  const [namespace, setNamespace] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  usePageRefresh(refresh);

  const namespaces = useMemo(
    () => [...new Set(skills.map((s) => s.namespace))].sort(),
    [skills]
  );

  const filtered = useMemo(() => {
    return skills.filter((s) => {
      if (search && !s.name.toLowerCase().includes(search.toLowerCase()) && !s.description?.toLowerCase().includes(search.toLowerCase())) return false;
      if (namespace && s.namespace !== namespace) return false;
      return true;
    });
  }, [skills, search, namespace]);

  const handleDelete = async (name: string) => {
    try {
      await deleteSkill(name);
      refresh();
    } catch {}
  };

  return (
    <div className="flex h-full flex-col">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.1, ease: "easeOut" }}
        className="flex-1 overflow-y-auto p-6"
      >
        <div className="flex items-center justify-between mb-4">
          <ListFilterBar
            search={search}
            onSearchChange={setSearch}
            searchPlaceholder="Search skills..."
            namespace={namespace}
            onNamespaceChange={setNamespace}
            namespaces={namespaces}
          />
          <div className="flex items-center gap-2 shrink-0 ml-4">
            <Button size="sm" onClick={() => setCreateOpen(true)} className="flex items-center gap-1.5">
              <Plus className="size-3 shrink-0" />
              <span>Create</span>
            </Button>
          </div>
        </div>

        {showLoading ? (
          <SkeletonTable />
        ) : loading ? (
          null
        ) : error ? (
          <div className="rounded-md border border-red-400/20 bg-red-400/5 p-4 text-sm text-red-400">
            {error}
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            title="No skills yet"
            description="Create a skill to attach reusable capabilities to your agents."
            action={{ label: "Create Skill", onClick: () => setCreateOpen(true) }}
          />
        ) : (
          <SkillCards skills={filtered} onDelete={handleDelete} />
        )}
      </motion.div>

      <CreateSkillModal open={createOpen} onOpenChange={setCreateOpen} onCreated={refresh} />
    </div>
  );
}
