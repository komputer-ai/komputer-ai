"use client";

import { useState, useMemo, useCallback } from "react";
import { motion } from "framer-motion";
import { Trash2, X, CheckSquare, Users } from "lucide-react";

import { Button } from "@/components/kit/button";
import { AgentCards, agentKey } from "@/components/agents/agent-cards";
import { EmptyState } from "@/components/shared/empty-state";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { ListFilterBar } from "@/components/shared/list-filter-bar";
import { useAgents } from "@/hooks/use-agents";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { usePageRefresh, usePageHeaderSlot } from "@/components/layout/app-shell";
import { deleteAgent, deleteSquad } from "@/lib/api";
import { SquadAwareDeleteDialog } from "@/components/shared/squad-aware-delete-dialog";
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

  async function handleDelete(name: string, namespace: string, opts?: { recreatePod?: boolean }) {
    try {
      await deleteAgent(name, namespace, opts);
      refresh();
    } catch {
      // Deletion errors are non-critical; next poll will update
    }
  }

  async function handleBulkDelete(opts?: { recreatePod?: boolean }) {
    if (selected.size === 0) return;
    setBulkDeleting(true);
    try {
      const targets = filtered.filter((a) => selected.has(agentKey(a)));
      await Promise.allSettled(targets.map((a) => deleteAgent(a.name, a.namespace, opts)));
      setSelected(new Set());
      refresh();
    } finally {
      setBulkDeleting(false);
    }
  }

  // Inject the bulk-action bar into the global header (left of "+ New Agent").
  const allSelected = selected.size > 0 && selected.size === filtered.length;

  // List the selected squad-member agents (and their squad names) so the user
  // can confirm exactly what will happen to them.
  const squadSelectedAgents = useMemo(() => {
    if (selected.size === 0) return [] as { name: string; namespace: string; squadName?: string }[];
    return filtered
      .filter((a) => selected.has(agentKey(a)) && a.squad)
      .map((a) => ({ name: a.name, namespace: a.namespace, squadName: a.squadName }));
  }, [selected, filtered]);
  const squadSelectedCount = squadSelectedAgents.length;

  const [bulkDialogOpen, setBulkDialogOpen] = useState(false);

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
        {squadSelectedCount > 0 && (
          <span
            className="flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-[var(--color-brand-violet)]/15 text-[var(--color-brand-violet)] border border-[var(--color-brand-violet)]/30"
            title={`${squadSelectedCount} selected agent${squadSelectedCount === 1 ? " is" : "s are"} in a squad`}
          >
            <Users className="size-2.5" />
            {squadSelectedCount} in squad
          </span>
        )}
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
        <Button
          variant="destructive"
          size="sm"
          disabled={bulkDeleting}
          className="!h-6 !px-2.5 text-xs"
          onClick={() => setBulkDialogOpen(true)}
        >
          <Trash2 className="size-3" data-icon="inline-start" />
          {bulkDeleting ? "Deleting..." : `Delete ${selected.size}`}
        </Button>
      </motion.div>
    );
  }, [selected.size, squadSelectedCount, bulkDeleting, clearSelection, selectAll, allSelected]);
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

      <SquadAwareDeleteDialog
        open={bulkDialogOpen}
        onOpenChange={setBulkDialogOpen}
        title={`Delete ${selected.size} agent${selected.size === 1 ? "" : "s"}?`}
        description="This will permanently delete the selected agents and their workspaces."
        squadMembers={squadSelectedAgents.map((a) => ({
          agentName: a.name,
          squad: { name: a.squadName ?? "", namespace: a.namespace },
        }))}
        confirmLabel={`Delete ${selected.size}`}
        onConfirm={async ({ recreatePod, deleteSquads }) => {
          if (deleteSquads) {
            const uniqueSquads = new Map<string, { name: string; namespace: string }>();
            for (const a of squadSelectedAgents) {
              if (a.squadName) {
                uniqueSquads.set(`${a.namespace}/${a.squadName}`, { name: a.squadName, namespace: a.namespace });
              }
            }
            await Promise.allSettled(
              Array.from(uniqueSquads.values()).map((s) => deleteSquad(s.name, s.namespace)),
            );
          }
          await handleBulkDelete({ recreatePod });
        }}
      />
    </div>
  );
}
