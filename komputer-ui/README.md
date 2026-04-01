# komputer-ui

Web dashboard for the komputer.ai platform. Provides a visual interface for monitoring and managing agents, offices, schedules, memories, and costs.

Built with [Next.js](https://nextjs.org/) 16, [React](https://react.dev/) 19, [Tailwind CSS](https://tailwindcss.com/) 4, and [Framer Motion](https://motion.dev/).

## Pages

| Page | Path | Description |
|------|------|-------------|
| Dashboard | `/` | Overview with stats (active agents, total cost, offices, schedules) and recent agents |
| Agents | `/agents` | List all agents with status, cost, model, and lifecycle |
| Agent Detail | `/agents/:name` | Agent details, live event stream, and actions (cancel, delete) |
| Memories | `/memories` | List reusable memories, create new ones, and upload `.md` or `.txt` files or folders |
| Memory Detail | `/memories/:name` | Edit memory content and description, inspect namespace and attached agents |
| Offices | `/offices` | List offices with member counts, active agents, and total cost |
| Office Detail | `/offices/:name` | Office members, per-agent status, and event history |
| Schedules | `/schedules` | List schedules with cron expression, run count, and cost |
| Schedule Detail | `/schedules/:name` | Schedule details and run history |
| Topology | `/topology` | Visual graph of manager-worker relationships across offices |
| Costs | `/costs` | Cost breakdown across agents and offices |

## Features

- **See every agent at a glance** — Dashboard shows active agents, total cost, offices, and schedules with live status updates
- **Create and manage agents from the browser** — Spin up new agents, send tasks, cancel runs, and delete resources without touching the CLI
- **Reusable memory management** — Create, edit, and delete memories, then bulk-import markdown or text files from a file picker or folder picker
- **Watch agents work in real-time** — Live event stream on agent detail pages shows thinking, tool calls, and results as they happen
- **Topology view** — Interactive graph visualizing manager-worker relationships across your offices
- **Cost visibility** — Track spend per agent, per office, and across the entire platform
- **Schedule management** — Create, monitor, and manage cron-based agent schedules with run history

## Configuration

The UI connects to the komputer-api via a runtime config file at `/config.js`. This allows changing the API URL without rebuilding the image.

**Default** (local development): connects to `http://localhost:8080`

**In Kubernetes**: Mount a ConfigMap to override `/config.js`:

```js
// config.js
window.__KOMPUTER_CONFIG__ = {
  apiUrl: "http://komputer-api.komputer-ai.svc.cluster.local:8080"
};
```

The API must have CORS enabled (it does by default).

## Development

### Prerequisites

- Node.js 20+

### Run locally

```bash
npm install
npm run dev
```

Opens at `http://localhost:3000`. Expects the komputer-api running on `http://localhost:8080`.

### Build

```bash
npm run build
```

### Docker

```bash
docker build -t komputer-ui:latest .
docker run -p 3000:3000 komputer-ui:latest
```

## Project Structure

```
komputer-ui/
├── src/
│   ├── app/                  # Next.js app router pages
│   │   ├── page.tsx          # Dashboard
│   │   ├── agents/           # Agent list + detail pages
│   │   ├── memories/         # Memory list + detail pages
│   │   ├── offices/          # Office list + detail pages
│   │   ├── schedules/        # Schedule list + detail pages
│   │   ├── topology/         # Visual agent graph
│   │   └── costs/            # Cost breakdown
│   ├── components/
│   │   ├── kit/              # Base UI components (Card, Button, etc.)
│   │   ├── layout/           # App shell, sidebar
│   │   ├── shared/           # Shared components (StatusBadge, etc.)
│   │   ├── agents/           # Agent-specific components
│   │   ├── offices/          # Office-specific components
│   │   ├── schedules/        # Schedule-specific components
│   │   ├── topology/         # Topology graph components
│   │   └── costs/            # Cost chart components
│   ├── hooks/                # Custom React hooks
│   └── lib/
│       ├── api.ts            # API client (agents, memories, offices, schedules, health)
│       ├── config.ts         # Runtime config loader
│       ├── types.ts          # TypeScript types
│       └── utils.ts          # Formatting helpers
├── public/
│   └── config.js             # Runtime API config (overridable via ConfigMap)
├── Dockerfile                # Multi-stage build (builder → standalone)
├── package.json
└── next.config.ts
```
