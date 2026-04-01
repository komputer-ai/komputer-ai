"use client";

import { useState, useMemo } from "react";
import { motion } from "framer-motion";

import { MemoryCards } from "@/components/memories/memory-cards";
import { CreateMemoryModal } from "@/components/memories/create-memory-modal";
import { EmptyState } from "@/components/shared/empty-state";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { ListFilterBar } from "@/components/shared/list-filter-bar";
import { useMemories } from "@/hooks/use-memories";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { usePageRefresh } from "@/components/layout/app-shell";
import { deleteMemory } from "@/lib/api";

export default function MemoriesPage() {
  const { memories, loading, error, refresh } = useMemories();
  const showLoading = useDelayedLoading(loading);
  const [search, setSearch] = useState("");
  const [namespace, setNamespace] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  usePageRefresh(refresh);

  const namespaces = useMemo(
    () => [...new Set(memories.map((m) => m.namespace))].sort(),
    [memories]
  );

  const filtered = useMemo(() => {
    return memories.filter((m) => {
      if (search && !m.name.toLowerCase().includes(search.toLowerCase()) && !m.description?.toLowerCase().includes(search.toLowerCase())) return false;
      if (namespace && m.namespace !== namespace) return false;
      return true;
    });
  }, [memories, search, namespace]);

  const handleDelete = async (name: string) => {
    try {
      await deleteMemory(name);
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
        <ListFilterBar
          search={search}
          onSearchChange={setSearch}
          searchPlaceholder="Search memories..."
          namespace={namespace}
          onNamespaceChange={setNamespace}
          namespaces={namespaces}
        />

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
            title="No memories yet"
            description="Create a memory to attach reusable knowledge to your agents."
            action={{ label: "Create Memory", onClick: () => setCreateOpen(true) }}
          />
        ) : (
          <MemoryCards memories={filtered} onDelete={handleDelete} />
        )}
      </motion.div>

      <CreateMemoryModal open={createOpen} onOpenChange={setCreateOpen} onCreated={refresh} />
    </div>
  );
}
