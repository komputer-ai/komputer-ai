"use client";

import Link from "next/link";
import { Trash2 } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/kit/table";
import { Button } from "@/components/kit/button";
import { StatusBadge } from "@/components/shared/status-badge";
import { CostBadge } from "@/components/shared/cost-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import type { OfficeResponse } from "@/lib/types";

type OfficeTableProps = {
  offices: OfficeResponse[];
  onDelete: (name: string) => void;
};

export function OfficeTable({ offices, onDelete }: OfficeTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Phase</TableHead>
          <TableHead>Manager</TableHead>
          <TableHead>Agents</TableHead>
          <TableHead>Active</TableHead>
          <TableHead>Completed</TableHead>
          <TableHead>Cost</TableHead>
          <TableHead>Created</TableHead>
          <TableHead className="w-10" />
        </TableRow>
      </TableHeader>
      <TableBody>
        <AnimatePresence initial={false}>
          {offices.map((office, i) => (
            <motion.tr
              key={office.name}
              className="border-b transition-colors hover:bg-muted/50"
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -6 }}
              transition={{ duration: 0.2, delay: i * 0.03 }}
            >
              <TableCell>
                <Link
                  href={`/offices/${office.name}`}
                  className="font-medium text-[var(--color-brand-blue)] hover:underline"
                >
                  {office.name}
                </Link>
              </TableCell>
              <TableCell>
                <StatusBadge status={office.phase} size="sm" />
              </TableCell>
              <TableCell>
                <Link
                  href={`/agents/${office.manager}`}
                  className="text-xs text-[var(--color-brand-blue)] hover:underline"
                >
                  {office.manager}
                </Link>
              </TableCell>
              <TableCell className="text-xs text-[var(--color-text-secondary)]">
                {office.totalAgents}
              </TableCell>
              <TableCell className="text-xs text-[var(--color-text-secondary)]">
                {office.activeAgents}
              </TableCell>
              <TableCell className="text-xs text-[var(--color-text-secondary)]">
                {office.completedAgents}
              </TableCell>
              <TableCell>
                <CostBadge cost={office.totalCostUSD} />
              </TableCell>
              <TableCell>
                <RelativeTime timestamp={office.createdAt} />
              </TableCell>
              <TableCell>
                <ConfirmDialog
                  title={`Delete ${office.name}?`}
                  description="This will permanently delete this office and all its member agents. This action cannot be undone."
                  onConfirm={() => onDelete(office.name)}
                  trigger={
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={`Delete ${office.name}`}
                    >
                      <Trash2 className="size-3.5 text-[var(--color-text-secondary)] hover:text-red-400" />
                    </Button>
                  }
                />
              </TableCell>
            </motion.tr>
          ))}
        </AnimatePresence>
      </TableBody>
    </Table>
  );
}
