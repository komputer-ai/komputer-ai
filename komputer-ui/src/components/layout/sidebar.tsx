"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import {
  LayoutDashboard,
  Bot,
  Building2,
  Clock,
  Network,
  DollarSign,
  Settings,
  PanelLeftClose,
  PanelLeftOpen,
} from "lucide-react";
import { Tooltip, TooltipProvider } from "@/components/kit/tooltip";

const navItems = [
  { label: "Dashboard", icon: LayoutDashboard, href: "/" },
  { label: "Agents", icon: Bot, href: "/agents" },
  { label: "Offices", icon: Building2, href: "/offices" },
  { label: "Schedules", icon: Clock, href: "/schedules" },
  { label: "Topology", icon: Network, href: "/topology" },
  { label: "Cost", icon: DollarSign, href: "/costs" },
];

const bottomItems = [
  { label: "Settings", icon: Settings, href: "/settings" },
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

  const link = (
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
        <span className="absolute left-0 top-0 bottom-0 w-[3px] rounded-r-full bg-gradient-to-b from-[var(--color-brand-blue-light)] to-[var(--color-brand-violet)] shadow-[0_0_8px_var(--color-brand-blue),0_0_16px_rgba(63,133,217,0.3)]" />
      )}
      <Icon className="h-5 w-5 shrink-0" style={isActive ? { color: "#5a9be6", filter: "drop-shadow(0 0 4px rgba(63,133,217,0.6))" } : undefined} />
      {!collapsed && (
        <motion.span
          className={isActive ? "bg-gradient-to-r from-[#5a9be6] to-[#A78BFA] bg-clip-text text-transparent" : ""}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.15 }}
        >
          {item.label}
        </motion.span>
      )}
    </Link>
  );

  if (collapsed) {
    return (
      <Tooltip content={item.label} side="right" sideOffset={8}>
        {link}
      </Tooltip>
    );
  }

  return link;
}

function CollapseButton({ collapsed, onClick }: { collapsed: boolean; onClick: () => void }) {
  const btn = (
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
  );

  if (collapsed) {
    return (
      <Tooltip content="Expand sidebar" side="right" sideOffset={8}>
        {btn}
      </Tooltip>
    );
  }

  return btn;
}

export function Sidebar() {
  const [collapsed, setCollapsed] = useState(false);
  const pathname = usePathname();

  return (
    <TooltipProvider>
      <motion.aside
        className="flex flex-col h-screen border-r border-[var(--color-border)] bg-[var(--color-bg-subtle)] shrink-0"
        animate={{ width: collapsed ? 56 : 210 }}
        transition={{ duration: 0.2, ease: "easeInOut" }}
      >
        {/* Logo */}
        <div className="flex items-center justify-between px-3 h-12 border-b border-[var(--color-border)]">
          <Link href="/" className="flex items-center gap-2 cursor-pointer hover:opacity-80 transition-opacity">
            <img src="/logo-no-bg.png" alt="komputer" width={34} height={18} className="shrink-0" style={{ filter: "drop-shadow(0 0 0.5px rgba(255,255,255,0.5))" }} />
            <AnimatePresence>
              {!collapsed && (
                <motion.div
                  className="overflow-hidden"
                  initial={{ opacity: 0, width: 0 }}
                  animate={{ opacity: 1, width: "auto" }}
                  exit={{ opacity: 0, width: 0 }}
                  transition={{ duration: 0.15 }}
                >
                  <img src="/logo-text-no-subtext-no-bg.png" alt="komputer" height={14} className="h-3.5 w-auto" style={{ filter: "drop-shadow(0 0 0.5px rgba(255,255,255,0.5))" }} />
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

        {/* Bottom: settings + expand button (when collapsed) */}
        <div className="border-t border-[var(--color-border)] px-2 py-2 flex flex-col gap-0.5">
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
