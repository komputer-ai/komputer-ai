"use client";

import { useState, useMemo, useCallback } from "react";
import { motion } from "framer-motion";
import { Trash2, X, CheckSquare } from "lucide-react";

import { Button } from "@/components/kit/button";
import { AgentCards, agentKey } from "@/components/agents/agent-cards";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { EmptyState } from "@/components/shared/empty-state";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { ListFilterBar } from "@/components/shared/list-filter-bar";
import { useAgents } from "@/hooks/use-agents";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { usePageRefresh, usePageHeaderSlot } from "@/components/layout/app-shell";
import { deleteAgent } from "@/lib/api";
import { cn } from "@/lib/utils";

const STATUS_FILTERS = ["All", "Running", "Sleeping", "Failed"] as const;
type StatusFilter = (typeof STATUS_FILTERS)[number];

export default function AgentsPage() {
  const { agents, loading, error, refresh } = useAgents();
  const showLoading = useDelayedLoading(loading);
  usePageRefresh(refresh);
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
    return result.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
  }, [agents, statusFilter, search, namespace]);

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [bulkDeleting, setBulkDeleting] = useState(false);

  const toggleSelect = useCallback((key: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }, []);

  const clearSelection = useCallback(() => setSelected(new Set()), []);

  const selectAll = useCallback(() => {
    setSelected(new Set(filtered.map((a) => agentKey(a))));
  }, [filtered]);

  async function handleDelete(name: string, namespace: string) {
    try {
      await deleteAgent(name, namespace);
      refresh();
    } catch {
      // Deletion errors are non-critical; next poll will update
    }
  }

  async function handleBulkDelete() {
    if (selected.size === 0) return;
    setBulkDeleting(true);
    try {
      const targets = filtered.filter((a) => selected.has(agentKey(a)));
      await Promise.allSettled(targets.map((a) => deleteAgent(a.name, a.namespace)));
      setSelected(new Set());
      refresh();
    } finally {
      setBulkDeleting(false);
    }
  }

  // Inject the bulk-action bar into the global header (left of "+ New Agent").
  const allSelected = selected.size > 0 && selected.size === filtered.length;

  const headerSlot = useMemo(() => {
    if (selected.size === 0) return null;
    return (
      <motion.div
        initial={{ opacity: 0, x: 8 }}
        animate={{ opacity: 1, x: 0 }}
        transition={{ duration: 0.12 }}
        className="flex items-center gap-1.5 pl-2.5 pr-1.5 py-1 rounded-full border border-[var(--color-border)] bg-[var(--color-surface)]"
      >
        <span className="text-xs text-[var(--color-text-secondary)]">{selected.size} selected</span>
        <div className="h-4 w-px bg-[var(--color-border)]" />
        <button
          type="button"
          onClick={selectAll}
          disabled={allSelected}
          className="flex items-center gap-1 h-6 px-2 rounded text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface-hover)] transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-default"
        >
          <CheckSquare className="size-3" />
          Select all
        </button>
        <button
          type="button"
          onClick={clearSelection}
          className="flex items-center gap-1 h-6 px-2 rounded text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface-hover)] transition-colors cursor-pointer"
        >
          <X className="size-3" />
          Clear
        </button>
        <ConfirmDialog
          title={`Delete ${selected.size} agent${selected.size === 1 ? "" : "s"}?`}
          description="This will permanently delete the selected agents and their workspaces."
          onConfirm={handleBulkDelete}
          trigger={
            <Button variant="destructive" size="sm" disabled={bulkDeleting} className="!h-6 !px-2.5 text-xs">
              <Trash2 className="size-3" data-icon="inline-start" />
              {bulkDeleting ? "Deleting..." : `Delete ${selected.size}`}
            </Button>
          }
        />
      </motion.div>
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected.size, bulkDeleting, clearSelection, selectAll, allSelected]);
  usePageHeaderSlot(headerSlot);

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
          <AgentCards
            agents={filtered}
            onDelete={handleDelete}
            selected={selected}
            onToggleSelect={toggleSelect}
          />
        )}
      </motion.div>
    </div>
  );
}
