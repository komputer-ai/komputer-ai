"use client";

import { useState, useCallback, useEffect, useMemo } from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";
import { ChevronLeft } from "lucide-react";
import { Sidebar } from "./sidebar";
import { HeaderAction } from "@/components/shared/header-action";
import { CreateAgentModal } from "@/components/agents/create-agent-modal";
import { CreateScheduleModal } from "@/components/schedules/create-schedule-modal";
import {
  CreateAgentModalContext,
  type AgentTemplate,
} from "@/lib/create-agent-modal-context";

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
  const [agentInitialValues, setAgentInitialValues] = useState<AgentTemplate | null>(null);
  const pathname = usePathname();
  const title = getPageTitle(pathname);
  const isSchedulesPage = pathname === "/schedules" || pathname.startsWith("/schedules/");

  const backLink = useMemo(() => {
    const segments = pathname.split("/").filter(Boolean);
    if (segments.length < 2) return null;
    const parentPath = `/${segments[0]}`;
    const parentTitle = pageTitles[parentPath];
    if (!parentTitle) return null;
    return { href: parentPath, label: parentTitle };
  }, [pathname]);

  useEffect(() => {
    document.title = title ? `${title} · Komputer.AI` : "Komputer.AI";
  }, [title]);

  const openWithTemplate = useCallback((template: AgentTemplate) => {
    setAgentInitialValues(template);
    setCreateAgentOpen(true);
  }, []);

  const handleAgentOpenChange = (open: boolean) => {
    setCreateAgentOpen(open);
    if (!open) setAgentInitialValues(null);
  };

  return (
    <CreateAgentModalContext.Provider value={{ openWithTemplate }}>
      <Sidebar />
      <div className="flex-1 flex flex-col overflow-hidden">
        <header className="flex items-center justify-between px-6 h-12 border-b border-[var(--color-border)] bg-[var(--color-bg-subtle)] shrink-0">
          <div className="flex items-center gap-2">
            {backLink && (
              <>
                <Link
                  href={backLink.href}
                  className="flex items-center gap-0.5 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
                >
                  <ChevronLeft className="h-3.5 w-3.5" />
                  {backLink.label}
                </Link>
                <span className="text-[var(--color-text-muted)] text-xs">/</span>
              </>
            )}
            <h1 className="text-[15px] font-semibold text-[var(--color-text)]">
              {title}
            </h1>
          </div>
          <div className="flex items-center gap-2">
            {isSchedulesPage ? (
              <HeaderAction label="New Schedule" onClick={() => setCreateScheduleOpen(true)} />
            ) : (
              <HeaderAction label="New Agent" onClick={() => { setAgentInitialValues(null); setCreateAgentOpen(true); }} />
            )}
          </div>
        </header>
        <main className="flex-1 overflow-y-auto">{children}</main>
      </div>
      <CreateAgentModal open={createAgentOpen} onOpenChange={handleAgentOpenChange} initialValues={agentInitialValues} />
      <CreateScheduleModal open={createScheduleOpen} onOpenChange={setCreateScheduleOpen} />
    </CreateAgentModalContext.Provider>
  );
}
