"use client";

import { useState, useMemo } from "react";
import { motion } from "framer-motion";
import { OfficeCards } from "@/components/offices/office-cards";
import { EmptyState } from "@/components/shared/empty-state";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { ListFilterBar } from "@/components/shared/list-filter-bar";
import { useOffices } from "@/hooks/use-offices";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { deleteOffice } from "@/lib/api";

export default function OfficesPage() {
  const { offices, loading, error, refresh } = useOffices();
  const showLoading = useDelayedLoading(loading);
  const [search, setSearch] = useState("");
  const [namespace, setNamespace] = useState("");

  const namespaces = useMemo(
    () => [...new Set(offices.map((o) => o.namespace))].sort(),
    [offices]
  );

  const filtered = useMemo(() => {
    let result = offices;
    if (namespace) {
      result = result.filter((o) => o.namespace === namespace);
    }
    if (search.trim()) {
      const q = search.trim().toLowerCase();
      result = result.filter((o) => o.name.toLowerCase().includes(q));
    }
    return result;
  }, [offices, search, namespace]);

  async function handleDelete(name: string) {
    try {
      await deleteOffice(name);
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
          searchPlaceholder="Search offices..."
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
            Failed to load offices: {error}
          </div>
        ) : filtered.length === 0 && offices.length === 0 ? (
          <EmptyState
            title="No offices yet"
            description="Offices are created automatically when a manager agent spawns sub-agents."
          />
        ) : filtered.length === 0 ? (
          <EmptyState
            title="No matching offices"
            description="Try adjusting your search or filter criteria."
          />
        ) : (
          <OfficeCards offices={filtered} onDelete={handleDelete} />
        )}
      </motion.div>
    </div>
  );
}
