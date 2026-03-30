"use client";

import { useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { cn } from "@/lib/utils";

interface TooltipProps {
  content: React.ReactNode;
  side?: "top" | "right" | "bottom" | "left";
  sideOffset?: number;
  children: React.ReactNode;
}

export function Tooltip({ content, side = "top", sideOffset = 6, children }: TooltipProps) {
  const [show, setShow] = useState(false);

  const positionClasses: Record<string, string> = {
    top: "bottom-full left-1/2 -translate-x-1/2",
    right: "left-full top-1/2 -translate-y-1/2",
    bottom: "top-full left-1/2 -translate-x-1/2",
    left: "right-full top-1/2 -translate-y-1/2",
  };

  const offsetStyle: React.CSSProperties = {
    ...(side === "top" && { marginBottom: sideOffset }),
    ...(side === "bottom" && { marginTop: sideOffset }),
    ...(side === "right" && { marginLeft: sideOffset }),
    ...(side === "left" && { marginRight: sideOffset }),
  };

  return (
    <div
      className="relative inline-flex"
      onMouseEnter={() => setShow(true)}
      onMouseLeave={() => setShow(false)}
    >
      {children}
      <AnimatePresence>
        {show && (
          <motion.div
            className={cn(
              "absolute z-50 px-2.5 py-1 text-[11px] font-medium rounded-[var(--radius-sm)]",
              "bg-[var(--color-surface-raised)] text-[var(--color-text)] border border-[var(--color-border)]",
              "shadow-[0_4px_12px_rgba(0,0,0,0.3)] whitespace-nowrap pointer-events-none",
              positionClasses[side]
            )}
            style={offsetStyle}
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.95 }}
            transition={{ duration: 0.1 }}
          >
            {content}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

// No-op wrapper for backward compatibility
export function TooltipProvider({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
