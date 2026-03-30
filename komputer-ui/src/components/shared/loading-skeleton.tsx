import { cn } from "@/lib/utils";

function Shimmer({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "animate-pulse rounded-md bg-[var(--color-surface)]",
        className
      )}
    />
  );
}

export function SkeletonCard() {
  return (
    <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
      <div className="flex items-center justify-between">
        <Shimmer className="h-4 w-32" />
        <Shimmer className="h-4 w-16" />
      </div>
      <Shimmer className="mt-3 h-3 w-48" />
      <Shimmer className="mt-2 h-3 w-24" />
    </div>
  );
}

export function SkeletonRow() {
  return (
    <tr>
      <td className="px-4 py-3">
        <Shimmer className="h-3 w-28" />
      </td>
      <td className="px-4 py-3">
        <Shimmer className="h-3 w-16" />
      </td>
      <td className="px-4 py-3">
        <Shimmer className="h-3 w-20" />
      </td>
      <td className="px-4 py-3">
        <Shimmer className="h-3 w-14" />
      </td>
    </tr>
  );
}

export function SkeletonTable() {
  return (
    <table className="w-full">
      <thead>
        <tr>
          {Array.from({ length: 4 }).map((_, i) => (
            <th key={i} className="px-4 py-3">
              <Shimmer className="h-3 w-20" />
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {Array.from({ length: 5 }).map((_, i) => (
          <SkeletonRow key={i} />
        ))}
      </tbody>
    </table>
  );
}
