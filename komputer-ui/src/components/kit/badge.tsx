import { cn } from "@/lib/utils";

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: "default" | "outline" | "secondary";
}

export function Badge({ className, variant = "default", ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 px-2 py-0.5 text-[11px] font-medium rounded-full",
        variant === "default" && "bg-[var(--color-brand-blue)]/10 text-[var(--color-brand-blue)] shadow-[inset_0_1px_0_rgba(255,255,255,0.04)]",
        variant === "outline" && "border border-[var(--color-border)] text-[var(--color-text-secondary)]",
        variant === "secondary" && "bg-[var(--color-surface-raised)] text-[var(--color-text-secondary)] shadow-[inset_0_1px_0_rgba(255,255,255,0.04)]",
        className
      )}
      {...props}
    />
  );
}
