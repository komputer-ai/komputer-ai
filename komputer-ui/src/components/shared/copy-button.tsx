"use client";

import { useState, useCallback } from "react";
import { Copy, Check } from "lucide-react";

/**
 * Legacy textarea + execCommand path for browsers / contexts where
 * `navigator.clipboard` is unavailable (notably non-localhost HTTP previews).
 * Exported so other copy UIs in the codebase can share the fallback.
 */
export function fallbackCopy(text: string, done: () => void) {
  const el = document.createElement("textarea");
  el.value = text;
  el.style.cssText = "position:fixed;top:-9999px;left:-9999px;opacity:0";
  document.body.appendChild(el);
  el.select();
  try { document.execCommand("copy"); done(); } catch {}
  document.body.removeChild(el);
}

type CopyButtonProps = {
  text: string;
  size?: "sm" | "md";
  className?: string;
};

export function CopyButton({ text, size = "sm", className = "" }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      const done = () => { setCopied(true); setTimeout(() => setCopied(false), 1500); };
      if (navigator.clipboard?.writeText) {
        navigator.clipboard.writeText(text).then(done).catch(() => fallbackCopy(text, done));
      } else {
        fallbackCopy(text, done);
      }
    },
    [text]
  );

  return (
    <button
      type="button"
      onClick={handleCopy}
      className={`shrink-0 rounded-md p-1 text-[var(--color-text-muted)] transition-colors hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-hover)] ${className}`}
      title="Copy to clipboard"
    >
      {copied ? (
        <Check className={`${size === "md" ? "size-3.5" : "size-3"} text-green-400`} />
      ) : (
        <Copy className={size === "md" ? "size-3.5" : "size-3"} />
      )}
    </button>
  );
}
