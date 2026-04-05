"use client";

import { motion } from "framer-motion";
import { useConnectorTemplates } from "@/hooks/use-connector-templates";
import type { ConnectorTemplate } from "@/lib/types";

type ServiceTemplateGridProps = {
  onSelect: (template: ConnectorTemplate) => void;
};

export function ServiceTemplateGrid({ onSelect }: ServiceTemplateGridProps) {
  const { templates, loading } = useConnectorTemplates();

  if (loading) {
    return <div className="py-12 text-sm text-[var(--color-text-muted)]">Loading...</div>;
  }

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-3">
      {templates.map((tpl, i) => (
        <motion.button
          key={tpl.service}
          type="button"
          initial={{ opacity: 0, y: 12, scale: 0.97 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{ duration: 0.25, delay: i * 0.04 }}
          disabled={!tpl.url}
          onClick={() => onSelect(tpl)}
          className="group relative flex flex-col items-center gap-3 rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-5 transition-all duration-200 hover:border-[var(--color-border-hover)] hover:shadow-[0_0_24px_rgba(139,92,246,0.08)] cursor-pointer text-left disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:border-[var(--color-border)]"
        >
          <div className="flex items-center justify-center w-12 h-12 rounded-xl transition-transform duration-200 group-hover:scale-110">
            <img src={tpl.logoUrl} alt={tpl.displayName} className="w-7 h-7" />
          </div>
          <div className="text-center">
            <p className="text-sm font-semibold text-[var(--color-text)]">{tpl.displayName}</p>
            <p className="mt-0.5 text-[11px] text-[var(--color-text-secondary)] line-clamp-2">
              {tpl.description}
            </p>
          </div>
          {!tpl.url && (
            <span className="absolute top-2 right-2 text-[8px] tracking-wider uppercase px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-400">
              coming soon
            </span>
          )}
        </motion.button>
      ))}
    </div>
  );
}
