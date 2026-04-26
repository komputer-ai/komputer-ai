"use client";

import { useState, useMemo } from "react";
import { motion } from "framer-motion";
import { SquadCards } from "@/components/squads/squad-cards";
import { EmptyState } from "@/components/shared/empty-state";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { ListFilterBar } from "@/components/shared/list-filter-bar";
import { useSquads } from "@/hooks/use-squads";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { usePageRefresh } from "@/components/layout/app-shell";
import { deleteSquad } from "@/lib/api";

export default function SquadsPage() {
  const { squads, loading, error, refresh } = useSquads();
  const showLoading = useDelayedLoading(loading);
  usePageRefresh(refresh);
  const [search, setSearch] = useState("");
  const [namespace, setNamespace] = useState("");

  const namespaces = useMemo(
    () => [...new Set(squads.map((s) => s.namespace))].sort(),
    [squads]
  );

  const filtered = useMemo(() => {
    let result = squads;
    if (namespace) {
      result = result.filter((s) => s.namespace === namespace);
    }
    if (search.trim()) {
      const q = search.trim().toLowerCase();
      result = result.filter((s) => s.name.toLowerCase().includes(q));
    }
    return result.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
  }, [squads, search, namespace]);

  async function handleDelete(name: string, ns: string) {
    try {
      await deleteSquad(name, ns);
      refresh();
    } catch {
      // Deletion errors are non-critical; next poll will update
    }
  }

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
          searchPlaceholder="Search squads..."
          namespace={namespace}
          onNamespaceChange={setNamespace}
          namespaces={namespaces}
        />

        {showLoading ? (
          <SkeletonTable />
        ) : loading ? (
          null
        ) : error ? (
          <div className="rounded-lg border border-red-400/20 bg-red-400/5 p-4 text-sm text-red-400">
            Failed to load squads: {error}
          </div>
        ) : filtered.length === 0 && squads.length === 0 ? (
          <EmptyState
            title="No squads yet"
            description="No squads yet. Create one by teaming up agents."
          />
        ) : filtered.length === 0 ? (
          <EmptyState
            title="No matching squads"
            description="Try adjusting your search or filter criteria."
          />
        ) : (
          <SquadCards squads={filtered} onDelete={handleDelete} />
        )}
      </motion.div>
    </div>
  );
}
