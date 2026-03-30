"use client";

import { useState, useMemo } from "react";
import { motion } from "framer-motion";
import { ScheduleCards } from "@/components/schedules/schedule-cards";
import { EmptyState } from "@/components/shared/empty-state";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { ListFilterBar } from "@/components/shared/list-filter-bar";
import { useSchedules } from "@/hooks/use-schedules";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { deleteSchedule } from "@/lib/api";

export default function SchedulesPage() {
  const { schedules, loading, error, refresh } = useSchedules();
  const showLoading = useDelayedLoading(loading);
  const [search, setSearch] = useState("");
  const [namespace, setNamespace] = useState("");

  const namespaces = useMemo(
    () => [...new Set(schedules.map((s) => s.namespace))].sort(),
    [schedules]
  );

  const filtered = useMemo(() => {
    let result = schedules;
    if (namespace) {
      result = result.filter((s) => s.namespace === namespace);
    }
    if (search.trim()) {
      const q = search.trim().toLowerCase();
      result = result.filter((s) => s.name.toLowerCase().includes(q));
    }
    return result;
  }, [schedules, search, namespace]);

  async function handleDelete(name: string) {
    try {
      await deleteSchedule(name);
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
          searchPlaceholder="Search schedules..."
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
            Failed to load schedules: {error}
          </div>
        ) : filtered.length === 0 && schedules.length === 0 ? (
          <EmptyState
            title="No schedules yet"
            description="Create your first schedule to run agents on a recurring basis."
          />
        ) : filtered.length === 0 ? (
          <EmptyState
            title="No matching schedules"
            description="Try adjusting your search or filter criteria."
          />
        ) : (
          <ScheduleCards schedules={filtered} onDelete={handleDelete} />
        )}
      </motion.div>
    </div>
  );
}
