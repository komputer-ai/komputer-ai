"use client";

import { useEffect, useState } from "react";
import { Moon, Sun } from "lucide-react";
import { motion } from "framer-motion";

const STORAGE_KEY = "komputer-theme";

type Theme = "dark" | "light";

/**
 * Sidebar-friendly theme toggle. Flips `data-theme` on <html> and persists
 * to localStorage. Visual treatment matches the sidebar's nav rows so it
 * sits naturally in the bottom region.
 */
export function ThemeToggle({ collapsed }: { collapsed: boolean }) {
  const [theme, setTheme] = useState<Theme>("dark");

  useEffect(() => {
    const stored = (typeof window !== "undefined" && window.localStorage.getItem(STORAGE_KEY)) as Theme | null;
    const initial: Theme = stored === "light" ? "light" : "dark";
    setTheme(initial);
    apply(initial);
  }, []);

  function toggle() {
    const next: Theme = theme === "dark" ? "light" : "dark";
    setTheme(next);
    apply(next);
    try {
      window.localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // ignore quota / private-mode errors
    }
  }

  const Icon = theme === "dark" ? Sun : Moon;
  const label = theme === "dark" ? "Light mode" : "Dark mode";

  return (
    <button
      type="button"
      onClick={toggle}
      aria-label={label}
      className="flex items-center gap-3 px-3 py-2 rounded-md text-[13px] font-medium font-[family-name:var(--font-sans)] text-[var(--color-text-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface)] transition-all duration-200 relative overflow-hidden cursor-pointer"
    >
      <Icon className="h-5 w-5 shrink-0" />
      {!collapsed && (
        <motion.span
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.15 }}
        >
          {label}
        </motion.span>
      )}
    </button>
  );
}

function apply(theme: Theme) {
  if (typeof document === "undefined") return;
  if (theme === "light") {
    document.documentElement.setAttribute("data-theme", "light");
  } else {
    document.documentElement.removeAttribute("data-theme");
  }
}
