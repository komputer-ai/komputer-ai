"use client";

/* eslint-disable @next/next/no-img-element */
import { useState } from "react";
import { Plug, Trash2, Users, ExternalLink } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { Button } from "@/components/kit/button";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { ConnectorDetailDialog } from "@/components/connectors/connector-detail-dialog";
import { ConnectorLogo } from "@/components/connectors/connector-logo";
import { formatRelativeTime } from "@/lib/utils";
import { useConnectorTemplates } from "@/hooks/use-connector-templates";
import type { ConnectorResponse } from "@/lib/types";

type ConnectorCardsProps = {
  connectors: ConnectorResponse[];
  onDelete: (name: string, namespace: string) => void;
};

export function ConnectorCards({ connectors, onDelete }: ConnectorCardsProps) {
  const { getByService } = useConnectorTemplates();
  const [selectedConnector, setSelectedConnector] = useState<ConnectorResponse | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  return (
    <>
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-2.5">
      <AnimatePresence>
        {connectors.map((conn, i) => {
          const tpl = getByService(conn.service);
          const color = tpl?.color ?? "#8899A6";
          return (
            <motion.div
              key={`${conn.namespace}/${conn.name}`}
              initial={{ opacity: 0, y: 12, scale: 0.97 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, scale: 0.97 }}
              transition={{ duration: 0.25, delay: i * 0.04 }}
              className="h-full"
            >
              <div
                className="group relative h-full min-h-32 overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] transition-all duration-200 hover:border-[var(--color-border-hover)] hover:shadow-[0_0_20px_rgba(139,92,246,0.06)] cursor-pointer"
                onClick={() => { setSelectedConnector(conn); setDetailOpen(true); }}
              >
                <div className="flex h-full flex-col p-3">
                  <div className="flex items-center gap-2">
                    <div className="flex items-center justify-center w-7 h-7 rounded-md shrink-0">
                      {tpl && tpl.logoUrl ? <ConnectorLogo src={tpl.logoUrl} alt={tpl.displayName} className="w-4 h-4" /> : <Plug className="w-3.5 h-3.5" style={{ color }} />}
                    </div>
                    <div className="flex flex-col min-w-0 flex-1">
                      <span className="text-[13px] font-semibold text-[var(--color-text)] truncate leading-tight">
                        {conn.name}
                      </span>
                      {tpl && (
                        <span className="text-[10px] text-[var(--color-text-muted)] truncate leading-tight">{tpl.displayName}</span>
                      )}
                    </div>
                    <div className="flex items-center gap-1.5 shrink-0">
                      <div onClick={(e) => e.stopPropagation()} className="opacity-0 group-hover:opacity-100 transition-opacity">
                        <ConfirmDialog
                          title={`Delete ${conn.name}?`}
                          description="This will remove this connector. Agents using it will lose access."
                          onConfirm={() => onDelete(conn.name, conn.namespace)}
                          trigger={
                            <Button variant="ghost" size="icon" className="h-5 w-5 p-0">
                              <Trash2 className="w-2.5 h-2.5 text-[var(--color-text-secondary)] hover:text-red-400 transition-colors" />
                            </Button>
                          }
                        />
                      </div>
                    </div>
                  </div>

                  <div className="mt-2 min-h-[2.75rem]">
                    <span className="inline-flex items-center text-[10px] tracking-wider px-1.5 py-0.5 rounded text-[var(--color-text-muted)] leading-none" style={{ backgroundColor: `${color}10`, color }}>
                      {conn.service}
                    </span>
                    {conn.url && (
                      <p className="mt-1 text-[10px] text-[var(--color-text-muted)] truncate flex items-center gap-0.5">
                        <ExternalLink className="w-2.5 h-2.5 shrink-0" />
                        {conn.url.replace(/^https?:\/\//, "")}
                      </p>
                    )}
                  </div>

                  <div className="mt-auto pt-3 space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">agents</span>
                      <span className="text-[11px] text-[var(--color-text-secondary)] flex items-center gap-1">
                        <Users className="w-2.5 h-2.5" />
                        {conn.attachedAgents}
                      </span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">age</span>
                      <span className="text-[11px] text-[var(--color-text-secondary)]">
                        {formatRelativeTime(conn.createdAt)}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </motion.div>
          );
        })}
      </AnimatePresence>
    </div>
    <ConnectorDetailDialog
      connector={selectedConnector}
      open={detailOpen}
      onOpenChange={setDetailOpen}
    />
    </>
  );
}
