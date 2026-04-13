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

  constructor(wsUrl: string, agentName: string) {
    this.agentName = agentName;
    this.ws = new WebSocket(`${wsUrl}/api/v1/agents/${agentName}/ws`);

    this.ws.onmessage = (event) => {
      const data = JSON.parse(event.data as string);
      const agentEvent: AgentEvent = {
        agentName: data.agentName || this.agentName,
        type: data.type || "",
        timestamp: data.timestamp || "",
        payload: data.payload || {},
      };

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
    };
  }

  close() {
    this.done = true;
    this.ws.close();
  }
}
