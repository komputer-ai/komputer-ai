import { formatCost } from "@/lib/utils";

type CostBadgeProps = {
  cost?: string;
};

export function CostBadge({ cost }: CostBadgeProps) {
  const formatted = formatCost(cost);
  const hasCost = formatted !== "\u2014";

  return (
    <span
      className={
        hasCost
          ? "font-mono text-xs text-[var(--color-text)]"
          : "text-xs text-[var(--color-text-secondary)]"
      }
    >
      {formatted}
    </span>
  );
}
