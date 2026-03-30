"use client";

import { useState } from "react";
import { usePathname } from "next/navigation";
import { Sidebar } from "./sidebar";
import { HeaderAction } from "@/components/shared/header-action";
import { CreateAgentModal } from "@/components/agents/create-agent-modal";
import { CreateScheduleModal } from "@/components/schedules/create-schedule-modal";

const pageTitles: Record<string, string> = {
  "/": "Dashboard",
  "/agents": "Agents",
  "/offices": "Offices",
  "/schedules": "Schedules",
  "/topology": "Topology",
  "/costs": "Cost",
  "/settings": "Settings",
};

function getPageTitle(pathname: string): string {
  if (pageTitles[pathname]) return pageTitles[pathname];
  const segments = pathname.split("/").filter(Boolean);
  if (segments.length >= 2) {
    const name = decodeURIComponent(segments[1]);
    if (segments[0] === "agents") return name;
    if (segments[0] === "offices") return name;
    if (segments[0] === "schedules") return name;
  }
  return "";
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const [createAgentOpen, setCreateAgentOpen] = useState(false);
  const [createScheduleOpen, setCreateScheduleOpen] = useState(false);
  const pathname = usePathname();
  const title = getPageTitle(pathname);
  const isSchedulesPage = pathname === "/schedules" || pathname.startsWith("/schedules/");

  return (
    <>
      <Sidebar />
      <div className="flex-1 flex flex-col overflow-hidden">
        <header className="flex items-center justify-between px-6 h-12 border-b border-[var(--color-border)] bg-[var(--color-bg-subtle)] shrink-0">
          <h1 className="text-[15px] font-semibold text-[var(--color-text)]">
            {title}
          </h1>
          <div className="flex items-center gap-2">
            {isSchedulesPage ? (
              <HeaderAction label="New Schedule" onClick={() => setCreateScheduleOpen(true)} />
            ) : (
              <HeaderAction label="New Agent" onClick={() => setCreateAgentOpen(true)} />
            )}
          </div>
        </header>
        <main className="flex-1 overflow-y-auto">{children}</main>
      </div>
      <CreateAgentModal open={createAgentOpen} onOpenChange={setCreateAgentOpen} />
      <CreateScheduleModal open={createScheduleOpen} onOpenChange={setCreateScheduleOpen} />
    </>
  );
}
