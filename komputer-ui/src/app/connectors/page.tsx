"use client";

import { useState, useMemo } from "react";
import { motion } from "framer-motion";

import { ConnectorCards } from "@/components/connectors/connector-cards";
import { CreateConnectorModal } from "@/components/connectors/create-connector-modal";
import { ServiceTemplateGrid } from "@/components/connectors/service-template-grid";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { ListFilterBar } from "@/components/shared/list-filter-bar";
import { useConnectors } from "@/hooks/use-connectors";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { usePageRefresh } from "@/components/layout/app-shell";
import { deleteConnector } from "@/lib/api";
import type { ConnectorTemplate } from "@/lib/types";

export default function ConnectorsPage() {
  const { connectors, loading, error, refresh } = useConnectors();
  const showLoading = useDelayedLoading(loading);
  const [search, setSearch] = useState("");
  const [namespace, setNamespace] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [initialTemplate, setInitialTemplate] = useState<ConnectorTemplate | undefined>();
  usePageRefresh(refresh);

  const namespaces = useMemo(
    () => [...new Set(connectors.map((c) => c.namespace))].sort(),
    [connectors]
  );

  const filtered = useMemo(() => {
    return connectors.filter((c) => {
      if (search && !c.name.toLowerCase().includes(search.toLowerCase()) && !c.service.toLowerCase().includes(search.toLowerCase()) && !c.displayName?.toLowerCase().includes(search.toLowerCase())) return false;
      if (namespace && c.namespace !== namespace) return false;
      return true;
    }).sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
  }, [connectors, search, namespace]);

  const handleDelete = async (name: string, namespace: string) => {
    try {
      await deleteConnector(name, namespace);
      refresh();
    } catch {}
  };

  const isEmpty = !loading && !error && connectors.length === 0;

  return (
    <div className="flex h-full flex-col">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.1, ease: "easeOut" }}
        className="flex-1 overflow-y-auto p-6"
      >
        {isEmpty ? (
          <div className="flex flex-col items-center">
            <div className="text-center mb-6">
              <h2 className="text-lg font-semibold text-[var(--color-text)]">Connect your agents</h2>
              <p className="mt-1 text-sm text-[var(--color-text-secondary)]">
                Choose a service to give your agents access to external tools via MCP.
              </p>
            </div>
            <ServiceTemplateGrid onSelect={(tpl) => { setInitialTemplate(tpl); setCreateOpen(true); }} />
          </div>
        ) : (
          <>
            <ListFilterBar
              search={search}
              onSearchChange={setSearch}
              searchPlaceholder="Search connectors..."
              namespace={namespace}
              onNamespaceChange={setNamespace}
              namespaces={namespaces}
            />

            {showLoading ? (
              <SkeletonTable />
            ) : error ? (
              <div className="rounded-md border border-red-400/20 bg-red-400/5 p-4 text-sm text-red-400">
                {error}
              </div>
            ) : (
              <ConnectorCards connectors={filtered} onDelete={handleDelete} />
            )}
          </>
        )}
      </motion.div>

      <CreateConnectorModal open={createOpen} onOpenChange={(v) => { setCreateOpen(v); if (!v) setInitialTemplate(undefined); }} onCreated={refresh} initialTemplate={initialTemplate} />
    </div>
  );
}
