import { Badge } from "@/components/kit/badge";
import { cn } from "@/lib/utils";

type StatusBadgeProps = {
  status: string;
  size?: "sm" | "md";
};

const statusColorMap: Record<string, { dot: string; text: string }> = {
  Running: { dot: "bg-[#34D399]", text: "text-[var(--color-text)]" },
  Active: { dot: "bg-[#34D399]", text: "text-[var(--color-text)]" },
  InProgress: { dot: "bg-[#34D399]", text: "text-[var(--color-text)]" },
  Sleeping: { dot: "bg-[#FBBF24]", text: "text-[var(--color-text)]" },
  Suspended: { dot: "bg-[#FBBF24]", text: "text-[var(--color-text)]" },
  Failed: { dot: "bg-[#F87171]", text: "text-[var(--color-text)]" },
  Error: { dot: "bg-[#F87171]", text: "text-[var(--color-text)]" },
  Pending: { dot: "bg-[#FBBF24]", text: "text-[var(--color-text)]" },
  Succeeded: { dot: "bg-[#34D399]", text: "text-[var(--color-text)]" },
  Complete: { dot: "bg-[#34D399]", text: "text-[var(--color-text)]" },
};

const defaultColors = { dot: "bg-[var(--color-text-secondary)]", text: "text-[var(--color-text-secondary)]" };

export function StatusBadge({ status, size = "md" }: StatusBadgeProps) {
  const colors = statusColorMap[status] ?? defaultColors;
  const shouldPulse = ["Running", "Active", "InProgress"].includes(status);

  return (
    <Badge
      variant="outline"
      className={cn(
        "gap-1.5 border-transparent bg-transparent text-sm px-0 py-0",
        size === "sm" && "text-xs"
      )}
    >
      <span
        className={cn(
          "inline-block shrink-0 rounded-full",
          colors.dot,
          shouldPulse && "animate-pulse shadow-[0_0_4px_currentColor]"
        )}
        style={{ width: 7, height: 7 }}
      />
      <span className={colors.text}>{status}</span>
    </Badge>
  );
}
