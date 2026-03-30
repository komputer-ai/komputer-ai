"use client";

import { useEffect, useRef } from "react";
import type { AgentEvent } from "@/lib/types";
import { EventCard } from "@/components/shared/event-card";

type AgentEventsProps = {
  events: AgentEvent[];
};

export function AgentEvents({ events }: AgentEventsProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [events.length]);

  if (events.length === 0) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-[var(--color-text-secondary)]">
        No events yet. Events will appear here in real time.
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col overflow-y-auto p-4">
      <div className="flex flex-col gap-2">
        {events.map((event, i) => (
          <EventCard key={`${event.timestamp}-${i}`} event={event} />
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
