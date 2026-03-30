import { cn } from "@/lib/utils";

interface LabelProps extends React.LabelHTMLAttributes<HTMLLabelElement> {}

export function Label({ className, ...props }: LabelProps) {
  return (
    <label
      className={cn(
        "text-[11px] font-semibold uppercase tracking-[0.08em] text-[var(--color-text-secondary)]",
        className
      )}
      {...props}
    />
  );
}
