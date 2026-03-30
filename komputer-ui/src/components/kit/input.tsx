import { forwardRef } from "react";
import { cn } from "@/lib/utils";

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, ...props }, ref) => (
    <input
      ref={ref}
      className={cn(
        "w-full h-8 px-3 rounded-[var(--radius-sm)] text-[13px] font-[family-name:var(--font-mono)]",
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
Input.displayName = "Input";
