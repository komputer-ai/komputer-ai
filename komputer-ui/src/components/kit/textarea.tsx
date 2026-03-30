import { forwardRef } from "react";
import { cn } from "@/lib/utils";

interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => (
    <textarea
      ref={ref}
      className={cn(
        "w-full min-h-20 px-3 py-2 rounded-[var(--radius-sm)] text-[13px] font-[family-name:var(--font-mono)] resize-y",
        "bg-[var(--color-bg)] text-[var(--color-text)]",
        "border border-[var(--color-border)]",
        "shadow-[inset_0_2px_4px_rgba(0,0,0,0.2)]",
        "placeholder:text-[var(--color-text-muted)]",
        "transition-all duration-150",
        "focus:outline-none focus:border-[var(--color-brand-blue)]/60 focus:shadow-[inset_0_2px_4px_rgba(0,0,0,0.2),0_0_0_2px_var(--color-brand-blue-glow)]",
        "hover:border-[var(--color-border-hover)]",
        "disabled:opacity-40 disabled:cursor-not-allowed",
        className
      )}
      {...props}
    />
  )
);
Textarea.displayName = "Textarea";
