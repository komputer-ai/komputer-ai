import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatCost(cost?: string): string {
  if (!cost) return '—';
  const num = parseFloat(cost);
  if (isNaN(num)) return '—';
  return `$${num.toFixed(4)}`;
}

export function formatRelativeTime(timestamp: string): string {
  const now = Date.now();
  const then = new Date(timestamp).getTime();
  const diff = now - then;

  if (diff < 0) {
    // Future time
    const absDiff = Math.abs(diff);
    if (absDiff < 60_000) return "just now";
    if (absDiff < 3_600_000) return `in ${Math.floor(absDiff / 60_000)}m`;
    if (absDiff < 86_400_000) return `in ${Math.floor(absDiff / 3_600_000)}h`;
    return `in ${Math.floor(absDiff / 86_400_000)}d`;
  }

  if (diff < 60_000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return `${Math.floor(diff / 86_400_000)}d ago`;
}

/**
 * The API returns Go duration strings ("30m0s", "24h0m0s"). Drop the zero-valued
 * trailing units so a value reads "30m" / "24h" instead.
 */
export function fmtDuration(d: string): string {
  return d.replace(/(\d+h)0m0s$/, "$1").replace(/(\d+m)0s$/, "$1") || d;
}

export function cronToHuman(cron: string): string {
  const parts = cron.split(' ');
  if (parts.length !== 5) return cron;
  const [min, hour, dom, month, dow] = parts;

  if (min.startsWith('*/')) return `Every ${min.slice(2)} minutes`;
  if (hour.startsWith('*/')) return `Every ${hour.slice(2)} hours`;
  if (dow === 'MON-FRI' && dom === '*' && month === '*') return `Weekdays at ${hour}:${min.padStart(2, '0')}`;
  if (dow === '*' && dom === '*' && month === '*') return `Daily at ${hour}:${min.padStart(2, '0')}`;
  if (dom === '1' && month === '*' && dow === '*') return `Monthly on the 1st at ${hour}:${min.padStart(2, '0')}`;

  return cron;
}
