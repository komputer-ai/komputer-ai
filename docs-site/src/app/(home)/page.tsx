import Link from 'next/link';
import type { LucideIcon } from 'lucide-react';
import { Boxes, Database, Radio, Users, BarChart3, Plug } from 'lucide-react';
import { DynamicCodeBlock } from 'fumadocs-ui/components/dynamic-codeblock';
import { getLatestRelease } from '@/lib/release';

export default async function HomePage() {
  const release = await getLatestRelease();
  return (
    <main className="flex flex-1 flex-col">
      {/* Hero */}
      <section className="relative overflow-hidden border-b border-fd-border">
        <div
          className="absolute inset-0 -z-10 opacity-40"
          style={{
            background:
              'radial-gradient(ellipse 80% 60% at 50% -10%, rgba(59,130,246,0.35), transparent), radial-gradient(ellipse 60% 50% at 80% 30%, rgba(139,92,246,0.25), transparent)',
          }}
        />
        <div className="mx-auto flex max-w-5xl flex-col items-center gap-6 px-6 py-24 text-center">
          <a
            href={release.url}
            className="rounded-full border border-fd-border bg-fd-card px-3 py-1 text-xs text-fd-muted-foreground transition hover:border-fd-primary hover:text-fd-foreground"
          >
            {release.name}
          </a>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src="/komputer-ai/logo-text-no-subtext-no-bg.png"
            alt="Komputer.AI"
            style={{ height: 56, width: 'auto' }}
          />
          <h1 className="text-balance text-5xl font-bold tracking-tight md:text-6xl">
            Distributed Claude AI agents on{' '}
            <span className="bg-gradient-to-r from-blue-500 to-violet-500 bg-clip-text text-transparent">
              Kubernetes
            </span>
          </h1>
          <p className="max-w-2xl text-balance text-lg text-fd-muted-foreground">
            A stateless, Kubernetes-native platform for running persistent Claude AI agents.
            Built on CRDs, operators, and the Kubernetes API — agents are first-class cluster resources.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-3">
            <Link
              href="/docs"
              className="rounded-md bg-fd-primary px-5 py-2.5 text-sm font-semibold text-fd-primary-foreground transition hover:opacity-90"
            >
              Read the docs →
            </Link>
            <a
              href="https://github.com/komputer-ai/komputer-ai"
              className="rounded-md border border-fd-border bg-fd-card px-5 py-2.5 text-sm font-semibold text-fd-foreground transition hover:bg-fd-accent"
            >
              View on GitHub
            </a>
          </div>

          {/* Product showcase: dashboard + CLI side-by-side
              Breaks out of the hero's max-w-5xl wrapper to take full viewport width. */}
          <div
            className="relative mt-12 grid items-start gap-6"
            style={{
              gridTemplateColumns: 'minmax(0, 1.7fr) minmax(0, 1fr)',
              width: 'min(1600px, calc(100vw - 2rem))',
            }}
          >
            <div
              className="absolute inset-0 -z-10 blur-3xl opacity-40"
              style={{
                background:
                  'linear-gradient(120deg, rgba(59,130,246,0.45), rgba(139,92,246,0.35))',
              }}
            />
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src="/komputer-ai/dashboard-page.png"
              alt="Komputer.AI dashboard"
              className="w-full rounded-xl border border-fd-border shadow-2xl"
            />
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src="/komputer-ai/demo.gif"
              alt="Komputer.AI CLI demo"
              width={900}
              height={500}
              className="w-full rounded-xl border border-fd-border shadow-2xl"
            />
          </div>
        </div>
      </section>

      {/* Python SDK quickstart — directly under the dashboard image */}
      <section className="border-b border-fd-border bg-fd-card/20">
        <div
          className="mx-auto w-full max-w-[1400px] items-start gap-10 px-4 py-20 md:grid md:px-6"
          style={{ gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1.2fr)' }}
        >
          <div>
            <span className="mb-3 inline-block rounded-full border border-fd-border bg-fd-card px-3 py-1 text-xs text-fd-muted-foreground">
              Python SDK
            </span>
            <h2 className="mb-4 text-3xl font-semibold tracking-tight">
              From <code className="rounded bg-fd-card px-1.5 py-0.5 text-xl font-mono">pip install</code> to streaming events in 10 lines
            </h2>
            <p className="mb-6 text-fd-muted-foreground">
              Create an agent, stream its tokens, tool calls, and cost — all over one WebSocket. Typed.
            </p>
            <div className="flex flex-wrap gap-3">
              <Link
                href="/docs/integration/overview"
                className="rounded-md bg-fd-primary px-4 py-2 text-sm font-semibold text-fd-primary-foreground transition hover:opacity-90"
              >
                Integration guide →
              </Link>
              <a
                href="https://github.com/komputer-ai/komputer-ai/tree/main/komputer-sdk/python"
                className="rounded-md border border-fd-border bg-fd-card px-4 py-2 text-sm font-semibold text-fd-foreground transition hover:bg-fd-accent"
              >
                SDK on GitHub
              </a>
            </div>
          </div>
          <div className="not-prose min-w-0">
            <DynamicCodeBlock lang="python" code={pythonExample} />
          </div>
        </div>
      </section>

      {/* Feature grid */}
      <section className="mx-auto w-full max-w-6xl px-6 py-20">
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {features.map((f) => (
            <FeatureCard key={f.title} {...f} />
          ))}
        </div>
      </section>

      {/* Agent page showcase */}
      <section className="border-t border-fd-border bg-fd-card/20">
        <div className="mx-auto grid w-full max-w-6xl items-center gap-10 px-6 py-20 lg:grid-cols-2">
          <div>
            <span className="mb-3 inline-block rounded-full border border-fd-border bg-fd-card px-3 py-1 text-xs text-fd-muted-foreground">
              Live agent view
            </span>
            <h2 className="mb-4 text-3xl font-semibold tracking-tight">
              Watch every agent think, in real time
            </h2>
            <p className="mb-6 max-w-md text-fd-muted-foreground">
              Stream token-by-token output, every tool call, and per-task cost over a single
              WebSocket. Same data the SDKs and CLI consume — no polling, no replay loss.
            </p>
            <Link
              href="/docs/integration/overview"
              className="text-sm font-semibold text-fd-primary hover:underline"
            >
              Read the integration guide →
            </Link>
          </div>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src="/komputer-ai/agent-page.png"
            alt="Komputer.AI agent detail view"
            className="w-full rounded-xl border border-fd-border shadow-2xl"
          />
        </div>
      </section>

      {/* Quick links */}
      <section className="border-t border-fd-border bg-fd-card/30">
        <div className="mx-auto w-full max-w-6xl px-6 py-16">
          <h2 className="mb-8 text-2xl font-semibold">Start here</h2>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {quickLinks.map((q) => (
              <Link
                key={q.title}
                href={q.href}
                className="group rounded-lg border border-fd-border bg-fd-card p-5 transition hover:border-fd-primary hover:bg-fd-accent"
              >
                <div className="mb-2 text-sm font-semibold text-fd-foreground">{q.title}</div>
                <p className="text-sm text-fd-muted-foreground">{q.desc}</p>
                <div className="mt-3 text-xs font-medium text-fd-primary opacity-0 transition group-hover:opacity-100">
                  Read →
                </div>
              </Link>
            ))}
          </div>
        </div>
      </section>
    </main>
  );
}

function FeatureCard({ title, desc, icon: Icon }: { title: string; desc: string; icon: LucideIcon }) {
  return (
    <div className="rounded-xl border border-fd-border bg-fd-card p-6 transition hover:border-fd-primary/50">
      <div className="mb-4 inline-flex h-10 w-10 items-center justify-center rounded-lg bg-fd-primary/10 text-fd-primary">
        <Icon className="h-5 w-5" strokeWidth={1.75} />
      </div>
      <h3 className="mb-2 text-base font-semibold">{title}</h3>
      <p className="text-sm leading-relaxed text-fd-muted-foreground">{desc}</p>
    </div>
  );
}

const features: { icon: LucideIcon; title: string; desc: string }[] = [
  {
    icon: Boxes,
    title: 'Kubernetes-native',
    desc: 'Agents are first-class CRDs. The operator reconciles them like any other resource — Pods, PVCs, lifecycle, GitOps.',
  },
  {
    icon: Database,
    title: 'Stateless API',
    desc: 'No database. CR `.status` is the source of truth; Redis is just transport. Restart anything, lose nothing.',
  },
  {
    icon: Radio,
    title: 'Real-time streaming',
    desc: 'REST for control, WebSocket for live event streams from every agent — token-by-token output, tool calls, costs.',
  },
  {
    icon: Users,
    title: 'Squads & sub-agents',
    desc: 'Multi-agent shared workspaces via co-located Pods. Manager agents create and orchestrate workers via MCP tools.',
  },
  {
    icon: BarChart3,
    title: 'Built-in observability',
    desc: 'Structured logs, Prometheus metrics, ServiceMonitors, and a Grafana dashboard ship with the chart.',
  },
  {
    icon: Plug,
    title: 'MCP connectors',
    desc: 'Plug in MCP servers per agent or template. Credentials managed via secrets, lifecycle by the operator.',
  },
];

const quickLinks = [
  { title: 'Installation', desc: 'Helm install on any Kubernetes cluster.', href: '/docs/getting-started/installation' },
  { title: 'Architecture', desc: 'How the operator, API, and agents fit together.', href: '/docs/architecture' },
  { title: 'Squads', desc: 'Co-located multi-agent pods.', href: '/docs/concepts/squads' },
  { title: 'Integration overview', desc: 'Drive komputer.ai with REST + WebSocket.', href: '/docs/integration/overview' },
];

const pythonExample = `# pip install komputer-ai-sdk
from komputer_ai.client import KomputerClient

client = KomputerClient("http://localhost:8080")

# Create an agent and give it a task
client.create_agent(
    name="my-agent",
    instructions="Analyze our K8s cluster and suggest cost optimizations",
    model="claude-sonnet-4-6",
)

# Stream events as the agent works
for event in client.watch_agent("my-agent"):
    if event.type == "text":
        print(event.payload.content)
    elif event.type == "tool_use":
        print(f"  -> using {event.payload.name}")
    elif event.type == "task_completed":
        print(f"Done — cost: \${event.payload.cost_usd}")
        break
`;
