/**
 * WebSocket streaming for agents — hand-written, preserved across regeneration.
 */

export interface AgentEvent {
  agentName: string;
  type: string;
  timestamp: string;
  payload: Record<string, any>;
}

/**
 * Async iterable stream of agent events over WebSocket.
 * Yields pre-fetched history events first, then live WebSocket events,
 * deduplicating by timestamp+type.
 *
 * @example
 * for await (const event of client.watchAgent("my-agent")) {
 *   if (event.type === "text") console.log(event.payload.content);
 *   if (event.type === "task_completed") break;
 * }
 */
export class AgentEventStream implements AsyncIterable<AgentEvent> {
  private ws: WebSocket;
  private agentName: string;
  private queue: AgentEvent[] = [];
  private resolve: ((value: IteratorResult<AgentEvent>) => void) | null = null;
  private done = false;
  private seen = new Set<string>();

  constructor(wsUrl: string, agentName: string, historyEvents?: AgentEvent[]) {
    this.agentName = agentName;

    // Seed dedup set and queue with history events.
    if (historyEvents) {
      for (const e of historyEvents) {
        const key = this.dedupKey(e);
        this.seen.add(key);
        this.queue.push(e);
      }
    }

    this.ws = new WebSocket(`${wsUrl}/api/v1/agents/${agentName}/ws`);

    this.ws.onmessage = (event) => {
      const data = JSON.parse(event.data as string);
      const agentEvent: AgentEvent = {
        agentName: data.agentName || this.agentName,
        type: data.type || "",
        timestamp: data.timestamp || "",
        payload: data.payload || {},
      };

      const key = this.dedupKey(agentEvent);
      if (this.seen.has(key)) return;
      this.seen.add(key);

      if (this.resolve) {
        const r = this.resolve;
        this.resolve = null;
        r({ value: agentEvent, done: false });
      } else {
        this.queue.push(agentEvent);
      }
    };

    this.ws.onclose = () => {
      this.done = true;
      if (this.resolve) {
        const r = this.resolve;
        this.resolve = null;
        r({ value: undefined as any, done: true });
      }
    };

    this.ws.onerror = () => {
      this.done = true;
      if (this.resolve) {
        const r = this.resolve;
        this.resolve = null;
        r({ value: undefined as any, done: true });
      }
    };
  }

  private dedupKey(e: AgentEvent): string {
    const normType = e.type === "task_started" ? "user_message" : e.type;
    return `${e.timestamp}:${normType}`;
  }

  [Symbol.asyncIterator](): AsyncIterator<AgentEvent> {
    return {
      next: () => {
        if (this.queue.length > 0) {
          return Promise.resolve({ value: this.queue.shift()!, done: false });
        }
        if (this.done) {
          return Promise.resolve({ value: undefined as any, done: true });
        }
        return new Promise((resolve) => {
          this.resolve = resolve;
        });
      },
      return: () => {
        this.close();
        return Promise.resolve({ value: undefined as any, done: true });
      },
    };
  }

  close() {
    this.done = true;
    this.ws.close();
  }
}
