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
import type { AgentResponse } from "@/lib/types";

type AgentTableProps = {
  agents: AgentResponse[];
  onDelete: (name: string) => void;
};

function lifecycleLabel(lifecycle?: string): string {
  if (!lifecycle) return "Default";
  if (lifecycle === "AutoDelete") return "Auto Delete";
  return lifecycle;
}

function taskLabel(agent: AgentResponse): string {
  if (!agent.taskStatus) return "--";
  if (agent.taskStatus === "InProgress") return "In Progress";
  return agent.taskStatus;
}

export function AgentTable({ agents, onDelete }: AgentTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Task</TableHead>
          <TableHead>Model</TableHead>
          <TableHead>Cost</TableHead>
          <TableHead>Lifecycle</TableHead>
          <TableHead>Created</TableHead>
          <TableHead className="w-10" />
        </TableRow>
      </TableHeader>
      <TableBody>
        <AnimatePresence initial={false}>
          {agents.map((agent, i) => (
            <motion.tr
              key={agent.name}
              className="border-b transition-colors hover:bg-muted/50"
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -6 }}
              transition={{ duration: 0.2, delay: i * 0.03 }}
            >
              <TableCell>
                <Link
                  href={`/agents/${agent.name}`}
                  className="font-medium text-[var(--color-brand-blue)] hover:underline"
                >
                  {agent.name}
                </Link>
              </TableCell>
              <TableCell>
                <StatusBadge status={agent.status} size="sm" />
              </TableCell>
              <TableCell className="text-xs text-[var(--color-text-secondary)]">
                {taskLabel(agent)}
              </TableCell>
              <TableCell className="text-xs text-[var(--color-text-secondary)]">
                {agent.model}
              </TableCell>
              <TableCell>
                <CostBadge cost={agent.totalCostUSD} />
              </TableCell>
              <TableCell className="text-xs text-[var(--color-text-secondary)]">
                {lifecycleLabel(agent.lifecycle)}
              </TableCell>
              <TableCell>
                <RelativeTime timestamp={agent.createdAt} />
              </TableCell>
              <TableCell>
                <ConfirmDialog
                  title={`Delete ${agent.name}?`}
                  description="This will permanently delete this agent and its workspace. This action cannot be undone."
                  onConfirm={() => onDelete(agent.name)}
                  trigger={
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={`Delete ${agent.name}`}
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
