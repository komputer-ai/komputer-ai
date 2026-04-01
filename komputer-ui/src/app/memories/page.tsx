"use client";

import { useState, useMemo, useRef, useEffect } from "react";
import { motion } from "framer-motion";

import { ChevronDown, FolderOpen, Plus, Upload } from "lucide-react";
import { Button } from "@/components/kit/button";
import { MemoryCards } from "@/components/memories/memory-cards";
import { CreateMemoryModal } from "@/components/memories/create-memory-modal";
import { EmptyState } from "@/components/shared/empty-state";
import { SkeletonTable } from "@/components/shared/loading-skeleton";
import { ListFilterBar } from "@/components/shared/list-filter-bar";
import { useMemories } from "@/hooks/use-memories";
import { useDelayedLoading } from "@/hooks/use-delayed-loading";
import { usePageRefresh } from "@/components/layout/app-shell";
import { deleteMemory, createMemory } from "@/lib/api";

type DirectoryInputProps = React.InputHTMLAttributes<HTMLInputElement> & {
  webkitdirectory?: string;
};

function isSupportedUpload(file: File) {
  return file.name.endsWith(".md") || file.name.endsWith(".txt");
}

function getMemoryName(file: File) {
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

export default function MemoriesPage() {
  const { memories, loading, error, refresh } = useMemories();
  const showLoading = useDelayedLoading(loading);
  const [search, setSearch] = useState("");
  const [namespace, setNamespace] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadMenuOpen, setUploadMenuOpen] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);
  const uploadMenuRef = useRef<HTMLDivElement>(null);
  usePageRefresh(refresh);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (!uploadMenuRef.current?.contains(event.target as Node)) {
        setUploadMenuOpen(false);
      }
    }

    if (uploadMenuOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [uploadMenuOpen]);

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files || files.length === 0) return;
    setUploadMenuOpen(false);
    setUploading(true);
    try {
      for (const file of Array.from(files)) {
        if (!isSupportedUpload(file)) continue;
        const content = await file.text();
        const name = getMemoryName(file);
        if (!name) continue;
        const description =
          "webkitRelativePath" in file && file.webkitRelativePath
            ? file.webkitRelativePath
            : file.name;
        await createMemory({ name, content, description });
      }
      refresh();
    } catch {}
    setUploading(false);
    if (fileInputRef.current) fileInputRef.current.value = "";
    if (folderInputRef.current) folderInputRef.current.value = "";
  };

  const namespaces = useMemo(
    () => [...new Set(memories.map((m) => m.namespace))].sort(),
    [memories]
  );

  const filtered = useMemo(() => {
    return memories.filter((m) => {
      if (search && !m.name.toLowerCase().includes(search.toLowerCase()) && !m.description?.toLowerCase().includes(search.toLowerCase())) return false;
      if (namespace && m.namespace !== namespace) return false;
      return true;
    });
  }, [memories, search, namespace]);

  const handleDelete = async (name: string) => {
    try {
      await deleteMemory(name);
      refresh();
    } catch {}
  };

  return (
    <div className="flex h-full flex-col">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.1, ease: "easeOut" }}
        className="flex-1 overflow-y-auto p-6"
      >
        <div className="flex items-center justify-between mb-4">
          <ListFilterBar
            search={search}
            onSearchChange={setSearch}
            searchPlaceholder="Search memories..."
            namespace={namespace}
            onNamespaceChange={setNamespace}
            namespaces={namespaces}
          />
          <div className="flex items-center gap-2 shrink-0 ml-4">
            <input
              ref={fileInputRef}
              type="file"
              accept=".md,.txt"
              multiple
              className="hidden"
              onChange={handleUpload}
            />
            <input
              {...({ webkitdirectory: "" } as DirectoryInputProps)}
              ref={folderInputRef}
              type="file"
              accept=".md,.txt"
              multiple
              className="hidden"
              onChange={handleUpload}
            />
            <div ref={uploadMenuRef} className="relative">
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setUploadMenuOpen((open) => !open)}
                disabled={uploading}
                className="flex items-center gap-1.5"
              >
                <Upload className="size-3 shrink-0" />
                <span>{uploading ? "Uploading..." : "Upload"}</span>
                {!uploading && <ChevronDown className="size-3 shrink-0" />}
              </Button>

              {uploadMenuOpen && !uploading && (
                <motion.div
                  initial={{ opacity: 0, y: -6, scale: 0.96 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, y: -4, scale: 0.98 }}
                  transition={{ duration: 0.16, ease: "easeOut" }}
                  className="absolute left-1/2 top-full z-20 mt-2 w-44 -translate-x-1/2 overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] shadow-lg"
                >
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    className="flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left text-sm text-[var(--color-text-primary)] transition-colors hover:bg-[var(--color-surface-hover)]"
                  >
                    <Upload className="size-3.5 shrink-0" />
                    <span>Upload files</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => folderInputRef.current?.click()}
                    className="flex w-full cursor-pointer items-center gap-2 border-t border-[var(--color-border)] px-3 py-2 text-left text-sm text-[var(--color-text-primary)] transition-colors hover:bg-[var(--color-surface-hover)]"
                  >
                    <FolderOpen className="size-3.5 shrink-0" />
                    <span>Upload folder</span>
                  </button>
                </motion.div>
              )}
            </div>
            <Button size="sm" onClick={() => setCreateOpen(true)} className="flex items-center gap-1.5">
              <Plus className="size-3 shrink-0" />
              <span>Create</span>
            </Button>
          </div>
        </div>

        {showLoading ? (
          <SkeletonTable />
        ) : loading ? (
          null
        ) : error ? (
          <div className="rounded-md border border-red-400/20 bg-red-400/5 p-4 text-sm text-red-400">
            {error}
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            title="No memories yet"
            description="Create a memory to attach reusable knowledge to your agents."
            action={{ label: "Create Memory", onClick: () => setCreateOpen(true) }}
          />
        ) : (
          <MemoryCards memories={filtered} onDelete={handleDelete} />
        )}
      </motion.div>

      <CreateMemoryModal open={createOpen} onOpenChange={setCreateOpen} onCreated={refresh} />
    </div>
  );
}
