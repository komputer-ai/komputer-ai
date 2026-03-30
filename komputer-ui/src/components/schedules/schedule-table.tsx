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
import { Badge } from "@/components/kit/badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { CostBadge } from "@/components/shared/cost-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { cronToHuman } from "@/lib/utils";
import type { ScheduleResponse } from "@/lib/types";

type ScheduleTableProps = {
  schedules: ScheduleResponse[];
  onDelete: (name: string) => void;
};

function runsLabel(schedule: ScheduleResponse): string {
  const success = schedule.successfulRuns ?? 0;
  const failed = schedule.failedRuns ?? 0;
  return `${success}/${failed}`;
}

export function ScheduleTable({ schedules, onDelete }: ScheduleTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Schedule</TableHead>
          <TableHead>Phase</TableHead>
          <TableHead>Agent</TableHead>
          <TableHead>Runs (ok/fail)</TableHead>
          <TableHead>Cost</TableHead>
          <TableHead>Next Run</TableHead>
          <TableHead>Created</TableHead>
          <TableHead className="w-10" />
        </TableRow>
      </TableHeader>
      <TableBody>
        <AnimatePresence initial={false}>
          {schedules.map((schedule, i) => (
            <motion.tr
              key={schedule.name}
              className="border-b transition-colors hover:bg-muted/50"
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -6 }}
              transition={{ duration: 0.2, delay: i * 0.03 }}
            >
              <TableCell>
                <div className="flex items-center gap-2">
                  <Link
                    href={`/schedules/${schedule.name}`}
                    className="font-medium text-[var(--color-brand-blue)] hover:underline"
                  >
                    {schedule.name}
                  </Link>
                  {schedule.autoDelete && (
                    <Badge variant="secondary" className="text-[9px] px-1 py-0">
                      one-time
                    </Badge>
                  )}
                </div>
              </TableCell>
              <TableCell>
                <div className="flex flex-col">
                  <span className="font-mono text-xs text-[var(--color-text)]">
                    {schedule.schedule}
                  </span>
                  <span className="text-[10px] text-[var(--color-text-secondary)]">
                    {cronToHuman(schedule.schedule)}
                  </span>
                </div>
              </TableCell>
              <TableCell>
                <StatusBadge status={schedule.phase} size="sm" />
              </TableCell>
              <TableCell>
                {schedule.agentName ? (
                  <Link
                    href={`/agents/${schedule.agentName}`}
                    className="text-xs text-[var(--color-brand-blue)] hover:underline"
                  >
                    {schedule.agentName}
                  </Link>
                ) : (
                  <span className="text-xs text-[var(--color-text-secondary)]">
                    --
                  </span>
                )}
              </TableCell>
              <TableCell className="font-mono text-xs text-[var(--color-text-secondary)]">
                {runsLabel(schedule)}
              </TableCell>
              <TableCell>
                <CostBadge cost={schedule.totalCostUSD} />
              </TableCell>
              <TableCell>
                {schedule.nextRunTime ? (
                  <RelativeTime timestamp={schedule.nextRunTime} />
                ) : (
                  <span className="text-xs text-[var(--color-text-secondary)]">
                    --
                  </span>
                )}
              </TableCell>
              <TableCell>
                <RelativeTime timestamp={schedule.createdAt} />
              </TableCell>
              <TableCell>
                <ConfirmDialog
                  title={`Delete ${schedule.name}?`}
                  description="This will permanently delete this schedule. This action cannot be undone."
                  onConfirm={() => onDelete(schedule.name)}
                  trigger={
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={`Delete ${schedule.name}`}
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
