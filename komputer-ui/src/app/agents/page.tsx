"use client";

import { useState, useMemo } from "react";
import { motion } from "framer-motion";

import { Button } from "@/components/kit/button";
import { AgentCards } from "@/components/agents/agent-cards";
import { EmptyState } from "@/components/shared/empty-state";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { ListFilterBar } from "@/components/shared/list-filter-bar";
import { useAgents } from "@/hooks/use-agents";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { deleteAgent } from "@/lib/api";
import { cn } from "@/lib/utils";

const STATUS_FILTERS = ["All", "Running", "Sleeping", "Failed"] as const;
type StatusFilter = (typeof STATUS_FILTERS)[number];

export default function AgentsPage() {
  const { agents, loading, error, refresh } = useAgents();
  const showLoading = useDelayedLoading(loading);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("All");
  const [search, setSearch] = useState("");
  const [namespace, setNamespace] = useState("");

  const namespaces = useMemo(
    () => [...new Set(agents.map((a) => a.namespace))].sort(),
    [agents]
  );

  const filtered = useMemo(() => {
    let result = agents;
    if (namespace) {
      result = result.filter((a) => a.namespace === namespace);
    }
    if (statusFilter !== "All") {
      result = result.filter((a) => a.status === statusFilter);
    }
    if (search.trim()) {
      const q = search.trim().toLowerCase();
      result = result.filter((a) => a.name.toLowerCase().includes(q));
    }
    return result;
  }, [agents, statusFilter, search, namespace]);

  async function handleDelete(name: string) {
    try {
      await deleteAgent(name);
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
          searchPlaceholder="Search agents..."
          namespace={namespace}
          onNamespaceChange={setNamespace}
          namespaces={namespaces}
        >
          <div className="flex gap-1">
            {STATUS_FILTERS.map((f) => (
              <Button
                key={f}
                size="sm"
                variant={statusFilter === f ? "primary" : "ghost"}
                onClick={() => setStatusFilter(f)}
                className={cn(
                  "text-xs",
                  statusFilter === f
                    ? "bg-[var(--color-brand-blue)]/15 text-[var(--color-brand-blue)]"
                    : "text-[var(--color-text-secondary)]"
                )}
              >
                {f}
              </Button>
            ))}
          </div>
        </ListFilterBar>

        {/* Content */}
        {showLoading ? (
          <SkeletonTable />
        ) : loading ? (
          null
        ) : error ? (
          <p className="text-sm text-red-400">{error}</p>
        ) : filtered.length === 0 && agents.length === 0 ? (
          <EmptyState
            title="No agents yet"
            description="Create your first agent to get started."
          />
        ) : filtered.length === 0 ? (
          <EmptyState
            title="No matching agents"
            description="Try adjusting your search or filter criteria."
          />
        ) : (
          <AgentCards agents={filtered} onDelete={handleDelete} />
        )}
      </motion.div>
    </div>
  );
}
