"use client";

import { useState, useMemo } from "react";
import { motion } from "framer-motion";
import { Tooltip } from "@/components/kit/tooltip";
import { SecretCards } from "@/components/secrets/secret-cards";
import { CreateSecretModal } from "@/components/secrets/create-secret-modal";
import { EmptyState } from "@/components/shared/empty-state";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { ListFilterBar } from "@/components/shared/list-filter-bar";
import { useSecrets } from "@/hooks/use-secrets";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { usePageRefresh } from "@/components/layout/app-shell";
import { deleteSecretResource } from "@/lib/api";

export default function SecretsPage() {
  const [showAll, setShowAll] = useState(false);
  const { secrets, loading, error, refresh } = useSecrets(showAll);
  const showLoading = useDelayedLoading(loading);
  const [search, setSearch] = useState("");
  const [namespace, setNamespace] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  usePageRefresh(refresh);

  const namespaces = useMemo(
    () => [...new Set(secrets.map((s) => s.namespace))].sort(),
    [secrets]
  );

  const filtered = useMemo(() => {
    return secrets.filter((s) => {
      if (search && !s.name.toLowerCase().includes(search.toLowerCase())) return false;
      if (namespace && s.namespace !== namespace) return false;
      return true;
    }).sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
  }, [secrets, search, namespace]);

  const handleDelete = async (name: string, ns: string) => {
    try {
      await deleteSecretResource(name, ns);
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
          searchPlaceholder="Search secrets..."
          namespace={namespace}
          onNamespaceChange={setNamespace}
          namespaces={namespaces}
        >
          <Tooltip content="Show non-komputer managed secrets in the cluster" side="right">
            <button
              type="button"
              onClick={() => setShowAll((v) => !v)}
              className={`text-xs px-3 py-1.5 rounded-full border transition-colors cursor-pointer shrink-0 ${
                showAll
                  ? "border-amber-500/50 bg-amber-500/10 text-amber-400"
                  : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
              }`}
            >
              Show all
            </button>
          </Tooltip>
        </ListFilterBar>

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
            title="No secrets yet"
            description="Create a secret to store sensitive values for your agents."
            action={{ label: "Create Secret", onClick: () => setCreateOpen(true) }}
          />
        ) : (
          <SecretCards secrets={filtered} onDelete={handleDelete} onUpdated={refresh} />
        )}
      </motion.div>

      <CreateSecretModal open={createOpen} onOpenChange={setCreateOpen} onCreated={refresh} />
    </div>
  );
}
