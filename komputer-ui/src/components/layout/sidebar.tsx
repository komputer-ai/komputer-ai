"use client";

import { useState, useRef } from "react";
import { createPortal } from "react-dom";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { ThemeToggle } from "./theme-toggle";
import {
  LayoutDashboard,
  Bot,
  Building2,
  Clock,
  Network,
  DollarSign,
  Brain,
  Wand2,
  KeyRound,
  Plug,
  Settings,
  PanelLeftClose,
  PanelLeftOpen,
  Users,
} from "lucide-react";
import { TooltipProvider } from "@/components/kit/tooltip";

const navItems = [
  { label: "Dashboard", icon: LayoutDashboard, href: "/" },
  { label: "Agents", icon: Bot, href: "/agents" },
  { label: "Memories", icon: Brain, href: "/memories" },
  { label: "Skills", icon: Wand2, href: "/skills" },
  { label: "Secrets", icon: KeyRound, href: "/secrets" },
  { label: "Connectors", icon: Plug, href: "/connectors" },
  { label: "Offices", icon: Building2, href: "/offices" },
  { label: "Squads", icon: Users, href: "/squads" },
  { label: "Schedules", icon: Clock, href: "/schedules" },
  { label: "Topology", icon: Network, href: "/topology" },
  { label: "Cost", icon: DollarSign, href: "/costs" },
];

const bottomItems: typeof navItems = [
  // { label: "Settings", icon: Settings, href: "/settings" },
];

function NavItem({
  item,
  isActive,
  collapsed,
}: {
  item: (typeof navItems)[0];
  isActive: boolean;
  collapsed: boolean;
}) {
  const Icon = item.icon;

  const [tooltipPos, setTooltipPos] = useState<{ top: number; left: number } | null>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  function handleMouseEnter() {
    if (!collapsed || !wrapperRef.current) return;
    const rect = wrapperRef.current.getBoundingClientRect();
    setTooltipPos({ top: rect.top + rect.height / 2, left: rect.right + 8 });
  }

  return (
    <div ref={wrapperRef} onMouseEnter={handleMouseEnter} onMouseLeave={() => setTooltipPos(null)}>
      <Link
        href={item.href}
        className={`
          flex items-center gap-3 px-3 py-2 rounded-md text-[13px] font-medium font-[family-name:var(--font-sans)] transition-all duration-200 relative overflow-hidden
          ${
            isActive
              ? "bg-[var(--color-brand-blue)]/10 shadow-[inset_0_0_12px_rgba(63,133,217,0.08)]"
              : "text-[var(--color-text-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface)]"
          }
        `}
      >
        {isActive && (
          <span className="absolute left-0 top-0 bottom-0 w-[3px] rounded-r-full bg-gradient-to-b from-[var(--color-brand-blue-light)] via-[var(--color-brand-violet)] to-[var(--color-brand-blue-light)] shadow-[0_0_8px_var(--color-brand-blue),0_0_16px_rgba(63,133,217,0.3)] animate-gradient" />
        )}
        <Icon className="h-5 w-5 shrink-0" style={isActive ? { color: "#5a9be6", filter: "drop-shadow(0 0 4px rgba(63,133,217,0.6))" } : undefined} />
        {!collapsed && (
          <motion.span
            className={isActive ? "bg-gradient-to-r from-[#5a9be6] via-[#A78BFA] to-[#5a9be6] bg-clip-text text-transparent animate-gradient" : ""}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.15 }}
          >
            {item.label}
          </motion.span>
        )}
      </Link>
      {tooltipPos && typeof document !== "undefined" && createPortal(
        <motion.div
          className="fixed z-[9999] px-2.5 py-1 text-[11px] font-medium rounded-[var(--radius-sm)] bg-[var(--color-surface-raised)] text-[var(--color-text)] border border-[var(--color-border)] shadow-[0_4px_12px_rgba(0,0,0,0.3)] whitespace-nowrap pointer-events-none -translate-y-1/2"
          style={{ top: tooltipPos.top, left: tooltipPos.left }}
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0.95 }}
          transition={{ duration: 0.1 }}
        >
          {item.label}
        </motion.div>,
        document.body
      )}
    </div>
  );
}

function CollapseButton({ collapsed, onClick }: { collapsed: boolean; onClick: () => void }) {
  const [tooltipPos, setTooltipPos] = useState<{ top: number; left: number } | null>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  function handleMouseEnter() {
    if (!collapsed || !wrapperRef.current) return;
    const rect = wrapperRef.current.getBoundingClientRect();
    setTooltipPos({ top: rect.top + rect.height / 2, left: rect.right + 8 });
  }

  return (
    <div ref={wrapperRef} onMouseEnter={handleMouseEnter} onMouseLeave={() => setTooltipPos(null)}>
      <button
        onClick={onClick}
        className="p-1.5 rounded text-[var(--color-text-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface)] transition-colors cursor-pointer"
      >
        {collapsed ? (
          <PanelLeftOpen className="h-4 w-4" />
        ) : (
          <PanelLeftClose className="h-4 w-4" />
        )}
      </button>
      {tooltipPos && typeof document !== "undefined" && createPortal(
        <motion.div
          className="fixed z-[9999] px-2.5 py-1 text-[11px] font-medium rounded-[var(--radius-sm)] bg-[var(--color-surface-raised)] text-[var(--color-text)] border border-[var(--color-border)] shadow-[0_4px_12px_rgba(0,0,0,0.3)] whitespace-nowrap pointer-events-none -translate-y-1/2"
          style={{ top: tooltipPos.top, left: tooltipPos.left }}
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0.95 }}
          transition={{ duration: 0.1 }}
        >
          Expand sidebar
        </motion.div>,
        document.body
      )}
    </div>
  );
}

export function Sidebar() {
  const [collapsed, setCollapsed] = useState(false);
  const pathname = usePathname();

  return (
    <TooltipProvider>
      <motion.aside
        className="flex flex-col h-screen border-r border-[var(--color-border)] bg-[var(--color-bg-subtle)] shrink-0"
        initial={false}
        animate={{ width: collapsed ? 56 : 210 }}
        transition={{ duration: 0.2, ease: "easeInOut" }}
      >
        {/* Logo */}
        <div className="flex items-center justify-between px-3 h-12 border-b border-[var(--color-border)]">
          <Link href="/" className="flex items-center gap-2 cursor-pointer hover:opacity-80 transition-opacity">
            <img src="/logo-no-bg.png" alt="komputer" width={34} height={18} className="shrink-0"  />
            <AnimatePresence>
              {!collapsed && (
                <motion.div
                  className="overflow-hidden"
                  initial={{ opacity: 0, width: 0 }}
                  animate={{ opacity: 1, width: "auto" }}
                  exit={{ opacity: 0, width: 0 }}
                  transition={{ duration: 0.15 }}
                >
                  <img src="/logo-text-no-subtext-no-bg.png" alt="komputer" height={14} className="h-3.5 w-auto"  />
                </motion.div>
              )}
            </AnimatePresence>
          </Link>
          {!collapsed && (
            <CollapseButton collapsed={false} onClick={() => setCollapsed(true)} />
          )}
        </div>

        {/* Expand button — right below logo when collapsed */}
        {collapsed && (
          <div className="flex justify-center py-2 border-b border-[var(--color-border)]">
            <CollapseButton collapsed={true} onClick={() => setCollapsed(false)} />
          </div>
        )}

        {/* Main nav */}
        <nav className="flex-1 flex flex-col gap-0.5 px-2 py-3 overflow-y-auto">
          {navItems.map((item) => (
            <NavItem
              key={item.href}
              item={item}
              isActive={
                item.href === "/"
                  ? pathname === "/"
                  : pathname.startsWith(item.href)
              }
              collapsed={collapsed}
            />
          ))}
        </nav>

        {/* Bottom: theme toggle + settings + expand button (when collapsed) */}
        <div className="border-t border-[var(--color-border)] px-2 py-2 flex flex-col gap-0.5">
          <ThemeToggle collapsed={collapsed} />
          {bottomItems.map((item) => (
            <NavItem
              key={item.href}
              item={item}
              isActive={pathname.startsWith(item.href)}
              collapsed={collapsed}
            />
          ))}
        </div>
      </motion.aside>
    </TooltipProvider>
  );
}
