"use client";

import { Search } from "lucide-react";
import { Input } from "@/components/kit/input";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/kit/select";

type ListFilterBarProps = {
  search: string;
  onSearchChange: (value: string) => void;
  searchPlaceholder?: string;
  namespace: string;
  onNamespaceChange: (value: string) => void;
  namespaces: string[];
  children?: React.ReactNode; // extra filter controls (e.g. status buttons)
};

export function ListFilterBar({
  search,
  onSearchChange,
  searchPlaceholder = "Search...",
  namespace,
  onNamespaceChange,
  namespaces,
  children,
}: ListFilterBarProps) {
  return (
    <div className="mb-4 flex flex-wrap items-center gap-3">
      {children}

      <div className="ml-auto flex items-center gap-2">
        {/* Namespace filter */}
        <Select value={namespace} onValueChange={onNamespaceChange}>
          <SelectTrigger className="h-7 w-40 text-xs">
            <SelectValue placeholder="All namespaces" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">All namespaces</SelectItem>
            {namespaces.map((ns) => (
              <SelectItem key={ns} value={ns}>
                {ns}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-[var(--color-text-secondary)]" />
          <Input
            placeholder={searchPlaceholder}
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            className="pl-7 h-7 w-48 text-xs bg-[var(--color-bg)] border-[var(--color-border)] text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
          />
        </div>
      </div>
    </div>
  );
}
