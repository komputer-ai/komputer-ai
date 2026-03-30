"use client";

import { useState, useCallback } from "react";
import { Copy, Check } from "lucide-react";

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
      navigator.clipboard.writeText(text).then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
      });
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
