"use client";

import Link from "next/link";
import { useState } from "react";
import { Crown, Wrench } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { formatCost } from "@/lib/utils";
import type { OfficeMemberResponse } from "@/lib/types";

type OfficeMembersProps = {
  members: OfficeMemberResponse[];
  manager: string;
  existingAgents?: Set<string>;
};

const taskColors: Record<string, string> = {
  InProgress: "#3f85d9",
  Complete: "#2DD4BF",
  Error: "#F87171",
};

export function OfficeMembersGrid({ members, manager, existingAgents }: OfficeMembersProps) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-2.5">
      {members.map((member) => {
        const isManager = member.name === manager;
        const isDeleted = existingAgents != null && !existingAgents.has(member.name);
        const RoleIcon = isManager ? Crown : Wrench;
        const accentColor = isManager ? "#3f85d9" : "#7c7c98";
        const taskColor = isDeleted ? "#F87171" : (taskColors[member.taskStatus ?? ""] ?? "#7c7c98");

        const card = (
          <div className="relative overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] transition-all duration-200 group-hover:border-[var(--color-border-hover)] group-hover:shadow-[0_0_20px_rgba(63,133,217,0.06),0_0_40px_rgba(139,92,246,0.04)]">
            <div className="p-3 flex flex-col h-full">
              {/* Status dot — top right */}
              <div className="absolute top-2.5 right-2.5">
                <span className="block w-2 h-2 rounded-full" style={{ backgroundColor: taskColor }} />
              </div>

              {/* Top: icon + name */}
              <div className="flex items-center gap-2">
                <RoleIcon className="w-3.5 h-3.5 shrink-0" style={{ color: accentColor }} />
                <span className="text-[13px] font-semibold text-[var(--color-text)] truncate pr-4">
                  {member.name}
                </span>
              </div>

              {/* Bottom */}
              <div className="mt-4 space-y-1">
                <div className="flex items-center justify-between">
                  <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">role</span>
                  <span className="text-[11px] text-[var(--color-text-secondary)]">{member.role}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">status</span>
                  <span className="text-[11px] text-[var(--color-text-secondary)]">
                    {isDeleted ? "Deleted" : (member.taskStatus || "idle")}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-muted)]">cost</span>
                  <span className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-text)]">{formatCost(member.lastTaskCostUSD)}</span>
                </div>
              </div>
            </div>
          </div>
        );

        if (isDeleted) {
          return (
            <DeletedMemberWrapper key={member.name}>
              {card}
            </DeletedMemberWrapper>
          );
        }

        return (
          <Link key={member.name} href={`/agents/${member.name}`} className="block group">
            {card}
          </Link>
        );
      })}
    </div>
  );
}

function DeletedMemberWrapper({ children }: { children: React.ReactNode }) {
  const [show, setShow] = useState(false);
  return (
    <div
      className="relative cursor-not-allowed"
      onMouseEnter={() => setShow(true)}
      onMouseLeave={() => setShow(false)}
    >
      {children}
      <AnimatePresence>
        {show && (
          <motion.div
            className="absolute z-50 left-1/2 -translate-x-1/2 top-full mt-1.5 px-2.5 py-1 text-[11px] font-medium rounded-[var(--radius-sm)] bg-[var(--color-surface-raised)] text-[var(--color-text)] border border-[var(--color-border)] shadow-[0_4px_12px_rgba(0,0,0,0.3)] whitespace-nowrap pointer-events-none"
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.95 }}
            transition={{ duration: 0.1 }}
          >
            This agent has been deleted
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
