// Humanizes agent status.conditions (see AgentCondition) for display.

// Known reasons get hand-written labels; anything else falls back to splitting
// the CamelCase reason into words so new operator reasons still read sensibly.
const REASON_LABELS: Record<string, string> = {
  SleepTTL: "Sleep TTL",
  ArchiveTTL: "Archive TTL",
  DeleteTTL: "Delete TTL",
  LifecycleSleep: "Lifecycle: Sleep",
  LifecycleArchive: "Lifecycle: Archive",
  SleepRequested: "Sleep requested",
  ArchiveRequested: "Archive requested",
  TaskTimeout: "Task timeout",
  TaskStarted: "Task started",
  Woken: "Woken",
  Archived: "Archived",
};

export function humanizeConditionReason(reason?: string): string {
  if (!reason) return "";
  if (REASON_LABELS[reason]) return REASON_LABELS[reason];
  // "SomeNewReason" -> "Some new reason"; keeps acronym runs ("TTL") together.
  const words = reason.replace(/([a-z0-9])([A-Z])/g, "$1 $2").replace(/([A-Z]+)([A-Z][a-z])/g, "$1 $2");
  return words.charAt(0).toUpperCase() + words.slice(1).toLowerCase().replace(/\b(ttl|api|s3)\b/g, (m) => m.toUpperCase());
}
