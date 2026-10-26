"use client";

import { useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { AnimatePresence, motion } from "framer-motion";
import { cn } from "@/lib/utils";

interface TooltipProps {
  content: React.ReactNode;
  side?: "top" | "right" | "bottom" | "left";
  sideOffset?: number;
  /** Follow the mouse instead of anchoring to the trigger (e.g. over a disabled area). */
  followCursor?: boolean;
  /** Classes for the trigger wrapper, e.g. to make it fill its parent. */
  className?: string;
  children: React.ReactNode;
}

// Keeps the tooltip this far from the viewport edges.
const EDGE = 8;

// The tooltip renders in a portal with fixed positioning, so no overflow/scroll
// ancestor can clip it. It's placed from the trigger's rect (or the cursor),
// flipped to the other side when it doesn't fit, then clamped to the viewport —
// all in a layout effect / direct style writes, so it never paints misplaced and
// following the cursor doesn't re-render on every mouse move.
export function Tooltip({ content, side = "top", sideOffset = 6, followCursor = false, className, children }: TooltipProps) {
  const [show, setShow] = useState(false);
  const triggerRef = useRef<HTMLDivElement>(null);
  const tipRef = useRef<HTMLDivElement>(null);
  const cursor = useRef({ x: 0, y: 0 });

  const place = () => {
    const tip = tipRef.current;
    const trigger = triggerRef.current;
    if (!tip || !trigger) return;
    const tw = tip.offsetWidth;
    const th = tip.offsetHeight;
    const vw = window.innerWidth;
    const vh = window.innerHeight;
    let left: number;
    let top: number;
    if (followCursor) {
      left = cursor.current.x + 14;
      top = cursor.current.y + 18;
      if (top + th > vh - EDGE) top = cursor.current.y - th - 10;
    } else {
      const r = trigger.getBoundingClientRect();
      if (side === "top" || side === "bottom") {
        left = r.left + r.width / 2 - tw / 2;
        top = side === "top" ? r.top - th - sideOffset : r.bottom + sideOffset;
        if (side === "top" && top < EDGE) top = r.bottom + sideOffset;
        if (side === "bottom" && top + th > vh - EDGE) top = r.top - th - sideOffset;
      } else {
        top = r.top + r.height / 2 - th / 2;
        left = side === "left" ? r.left - tw - sideOffset : r.right + sideOffset;
        if (side === "left" && left < EDGE) left = r.right + sideOffset;
        if (side === "right" && left + tw > vw - EDGE) left = r.left - tw - sideOffset;
      }
    }
    tip.style.left = `${Math.min(Math.max(EDGE, left), vw - tw - EDGE)}px`;
    tip.style.top = `${Math.min(Math.max(EDGE, top), vh - th - EDGE)}px`;
  };

  // Re-place on every render while shown: the content can change size.
  useLayoutEffect(() => {
    if (show) place();
  });

  return (
    <div
      ref={triggerRef}
      className={cn("relative inline-flex", className)}
      onMouseEnter={(e) => {
        cursor.current = { x: e.clientX, y: e.clientY };
        setShow(true);
      }}
      onMouseLeave={() => setShow(false)}
      onMouseMove={
        followCursor
          ? (e) => {
              cursor.current = { x: e.clientX, y: e.clientY };
              place();
            }
          : undefined
      }
    >
      {children}
      {typeof document !== "undefined" &&
        createPortal(
          <AnimatePresence>
            {show && (
              <motion.div
                ref={tipRef}
                role="tooltip"
                className={cn(
                  "fixed left-0 top-0 z-[60] px-2.5 py-1 text-[11px] font-medium rounded-[var(--radius-sm)]",
                  "bg-[var(--color-surface-raised)] text-[var(--color-text)] border border-[var(--color-border)]",
                  "shadow-[0_4px_12px_rgba(0,0,0,0.3)] w-max max-w-[360px] whitespace-normal break-words pointer-events-none",
                )}
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.95 }}
                transition={{ duration: 0.1 }}
              >
                {content}
              </motion.div>
            )}
          </AnimatePresence>,
          document.body,
        )}
    </div>
  );
}

// No-op wrapper for backward compatibility
export function TooltipProvider({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
