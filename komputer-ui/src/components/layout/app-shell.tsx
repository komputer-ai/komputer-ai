"use client";

import { useState, useCallback, useEffect, useMemo, useRef, createContext, useContext } from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, ChevronLeft, FolderOpen, RefreshCw, Upload } from "lucide-react";
import { Sidebar } from "./sidebar";
import { HeaderAction } from "@/components/shared/header-action";
import { CreateAgentModal } from "@/components/agents/create-agent-modal";
import { CreateScheduleModal } from "@/components/schedules/create-schedule-modal";
import { CreateMemoryModal } from "@/components/memories/create-memory-modal";
import { CreateSkillModal } from "@/components/skills/create-skill-modal";
import { CreateSecretModal } from "@/components/secrets/create-secret-modal";
import { CreateConnectorModal } from "@/components/connectors/create-connector-modal";
import { createMemory, createSkill } from "@/lib/api";
import {
  CreateAgentModalContext,
  type AgentTemplate,
} from "@/lib/create-agent-modal-context";

// --- Refresh context ---
type RefreshContextValue = {
  register: (fn: () => void) => void;
  unregister: () => void;
};
const RefreshContext = createContext<RefreshContextValue | null>(null);

export function usePageRefresh(refreshFn: () => void) {
  const ctx = useContext(RefreshContext);
  const fnRef = useRef(refreshFn);
  fnRef.current = refreshFn;
  useEffect(() => {
    ctx?.register(() => fnRef.current());
    return () => ctx?.unregister();
  }, [ctx]);
}

// --- Header slot context — lets a page inject custom content (e.g. bulk actions)
// to the left of the page-specific create button.
type HeaderSlotContextValue = {
  setSlot: (node: React.ReactNode | null) => void;
};
const HeaderSlotContext = createContext<HeaderSlotContextValue | null>(null);

export function usePageHeaderSlot(node: React.ReactNode | null) {
  const ctx = useContext(HeaderSlotContext);
  useEffect(() => {
    ctx?.setSlot(node);
    return () => ctx?.setSlot(null);
  }, [ctx, node]);
}

const pageTitles: Record<string, string> = {
  "/": "Dashboard",
  "/agents": "Agents",
  "/memories": "Memories",
  "/skills": "Skills",
  "/secrets": "Secrets",
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
    if (segments[0] === "memories") return name;
    if (segments[0] === "skills") return name;
  }
  return "";
}

type DirectoryInputProps = React.InputHTMLAttributes<HTMLInputElement> & {
  webkitdirectory?: string;
};

function isSupportedUpload(file: File) {
  return file.name.endsWith(".md") || file.name.endsWith(".txt");
}

function getUploadName(file: File) {
  const sourcePath =
    "webkitRelativePath" in file && file.webkitRelativePath
      ? file.webkitRelativePath
      : file.name;
  return sourcePath
    .replace(/\.(md|txt)$/i, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const [createAgentOpen, setCreateAgentOpen] = useState(false);
  const [createScheduleOpen, setCreateScheduleOpen] = useState(false);
  const [createMemoryOpen, setCreateMemoryOpen] = useState(false);
  const [createSkillOpen, setCreateSkillOpen] = useState(false);
  const [createSecretOpen, setCreateSecretOpen] = useState(false);
  const [createConnectorOpen, setCreateConnectorOpen] = useState(false);
  const [agentInitialValues, setAgentInitialValues] = useState<AgentTemplate | null>(null);
  const pathname = usePathname();
  const title = getPageTitle(pathname);

  const isSchedulesPage = pathname === "/schedules" || pathname.startsWith("/schedules/");
  const isMemoriesPage = pathname === "/memories" || pathname.startsWith("/memories/");
  const isSkillsPage = pathname === "/skills" || pathname.startsWith("/skills/");
  const isSecretsPage = pathname === "/secrets" || pathname.startsWith("/secrets/");
  const isConnectorsPage = pathname === "/connectors" || pathname.startsWith("/connectors/");

  const backLink = useMemo(() => {
    const segments = pathname.split("/").filter(Boolean);
    if (segments.length < 2) return null;
    const parentPath = `/${segments[0]}`;
    const parentTitle = pageTitles[parentPath];
    if (!parentTitle) return null;
    return { href: parentPath, label: parentTitle };
  }, [pathname]);

  const fullTitle = title ? `${title} · Komputer.AI` : "Komputer.AI";

  const [refreshFn, setRefreshFn] = useState<(() => void) | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const refreshCtx = useMemo<RefreshContextValue>(() => ({
    register: (fn) => setRefreshFn(() => fn),
    unregister: () => setRefreshFn(null),
  }), []);

  const [headerSlot, setHeaderSlot] = useState<React.ReactNode | null>(null);
  const headerSlotCtx = useMemo<HeaderSlotContextValue>(() => ({
    setSlot: (node) => setHeaderSlot(node),
  }), []);

  const handleRefresh = useCallback(() => {
    if (!refreshFn) return;
    setRefreshing(true);
    refreshFn();
    setTimeout(() => setRefreshing(false), 600);
  }, [refreshFn]);

  const openWithTemplate = useCallback((template: AgentTemplate) => {
    setAgentInitialValues(template);
    setCreateAgentOpen(true);
  }, []);

  const handleAgentOpenChange = (open: boolean) => {
    setCreateAgentOpen(open);
    if (!open) setAgentInitialValues(null);
  };

  // --- Upload support for memories/skills ---
  const [uploading, setUploading] = useState(false);
  const [uploadMenuOpen, setUploadMenuOpen] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);
  const uploadMenuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!uploadMenuOpen) return;
    function handleClick(e: MouseEvent) {
      if (!uploadMenuRef.current?.contains(e.target as Node)) setUploadMenuOpen(false);
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [uploadMenuOpen]);

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files || files.length === 0) return;
    setUploadMenuOpen(false);
    setUploading(true);
    const createFn = isSkillsPage ? createSkill : createMemory;
    try {
      for (const file of Array.from(files)) {
        if (!isSupportedUpload(file)) continue;
        const content = await file.text();
        const name = getUploadName(file);
        if (!name) continue;
        const description =
          "webkitRelativePath" in file && file.webkitRelativePath
            ? file.webkitRelativePath
            : file.name;
        await createFn({ name, content, description } as any);
      }
      refreshFn?.();
    } catch {}
    setUploading(false);
    if (fileInputRef.current) fileInputRef.current.value = "";
    if (folderInputRef.current) folderInputRef.current.value = "";
  };

  const showUpload = isMemoriesPage || isSkillsPage;

  return (
    <CreateAgentModalContext.Provider value={{ openWithTemplate }}>
      <title>{fullTitle}</title>
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
            {/* Upload button for memories/skills */}
            {showUpload && (
              <>
                <input ref={fileInputRef} type="file" accept=".md,.txt" multiple className="hidden" onChange={handleUpload} />
                <input {...({ webkitdirectory: "" } as DirectoryInputProps)} ref={folderInputRef} type="file" accept=".md,.txt" multiple className="hidden" onChange={handleUpload} />
                <div ref={uploadMenuRef} className="relative">
                  <button
                    onClick={() => setUploadMenuOpen((o) => !o)}
                    disabled={uploading}
                    className="flex items-center gap-1.5 h-7 px-2.5 rounded-md text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface-hover)] transition-colors cursor-pointer disabled:opacity-50"
                  >
                    <Upload className="size-3" />
                    <span>{uploading ? "Uploading..." : "Upload"}</span>
                    {!uploading && <ChevronDown className="size-3" />}
                  </button>
                  <AnimatePresence>
                    {uploadMenuOpen && !uploading && (
                      <motion.div
                        initial={{ opacity: 0, y: -4, scale: 0.96 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: -4, scale: 0.96 }}
                        transition={{ duration: 0.12 }}
                        className="absolute left-1/2 -translate-x-1/2 top-full z-20 mt-1 w-40 overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface-raised)] shadow-lg"
                      >
                        <button type="button" onClick={() => fileInputRef.current?.click()} className="flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left text-xs text-[var(--color-text)] transition-colors hover:bg-[var(--color-surface-hover)]">
                          <Upload className="size-3" /> Upload files
                        </button>
                        <button type="button" onClick={() => folderInputRef.current?.click()} className="flex w-full cursor-pointer items-center gap-2 border-t border-[var(--color-border)] px-3 py-2 text-left text-xs text-[var(--color-text)] transition-colors hover:bg-[var(--color-surface-hover)]">
                          <FolderOpen className="size-3" /> Upload folder
                        </button>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>
              </>
            )}

            {/* Page-injected header content (e.g. bulk actions) */}
            {headerSlot}

            {/* Page-specific create button */}
            {isSchedulesPage ? (
              <HeaderAction label="New Schedule" onClick={() => setCreateScheduleOpen(true)} />
            ) : isMemoriesPage ? (
              <HeaderAction label="New Memory" onClick={() => setCreateMemoryOpen(true)} />
            ) : isSkillsPage ? (
              <HeaderAction label="New Skill" onClick={() => setCreateSkillOpen(true)} />
            ) : isSecretsPage ? (
              <HeaderAction label="New Secret" onClick={() => setCreateSecretOpen(true)} />
            ) : isConnectorsPage ? (
              <HeaderAction label="New Connector" onClick={() => setCreateConnectorOpen(true)} />
            ) : (
              <HeaderAction label="New Agent" onClick={() => { setAgentInitialValues(null); setCreateAgentOpen(true); }} />
            )}

            {/* Refresh — always rightmost */}
            {refreshFn && (
              <button
                onClick={handleRefresh}
                className="flex size-7 items-center justify-center rounded-md text-[var(--color-text-secondary)] hover:text-[var(--color-brand-blue-light)] hover:bg-[var(--color-surface-hover)] transition-colors cursor-pointer"
                title="Refresh"
              >
                <RefreshCw className={`size-3.5 ${refreshing ? "animate-spin" : ""}`} />
              </button>
            )}
          </div>
        </header>
        <main className="flex-1 overflow-y-auto">
          <RefreshContext.Provider value={refreshCtx}>
            <HeaderSlotContext.Provider value={headerSlotCtx}>
              {children}
            </HeaderSlotContext.Provider>
          </RefreshContext.Provider>
        </main>
      </div>
      <CreateAgentModal open={createAgentOpen} onOpenChange={handleAgentOpenChange} initialValues={agentInitialValues} />
      <CreateScheduleModal open={createScheduleOpen} onOpenChange={setCreateScheduleOpen} />
      <CreateMemoryModal open={createMemoryOpen} onOpenChange={setCreateMemoryOpen} onCreated={() => refreshFn?.()} />
      <CreateSkillModal open={createSkillOpen} onOpenChange={setCreateSkillOpen} onCreated={() => refreshFn?.()} />
      <CreateSecretModal open={createSecretOpen} onOpenChange={setCreateSecretOpen} onCreated={() => refreshFn?.()} />
      <CreateConnectorModal open={createConnectorOpen} onOpenChange={setCreateConnectorOpen} onCreated={() => refreshFn?.()} />
    </CreateAgentModalContext.Provider>
  );
}
