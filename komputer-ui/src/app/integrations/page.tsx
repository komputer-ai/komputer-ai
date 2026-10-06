"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import { AlertTriangle, ExternalLink } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/kit/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/kit/tabs";
import { Badge } from "@/components/kit/badge";
import { CopyButton } from "@/components/shared/copy-button";
import { getConfig } from "@/lib/config";
import { checkHealth } from "@/lib/api";
import {
  mcpEndpointUrl,
  isLocalUrl,
  claudeCodeAddCommand,
  claudeDesktopConfig,
  type ClaudeCodeScope,
} from "@/lib/mcp-connect";

type HealthState = "checking" | "reachable" | "unreachable";

const MCP_DOCS_URL = "https://komputer-ai.github.io/komputer-ai/docs/integration/mcp-server";

const EXAMPLE_PROMPTS = [
  "List my komputer agents and their status",
  "Create an agent named hello that replies 'pong', wait for it to finish, then delete it",
];

const SCOPE_OPTIONS: { value: ClaudeCodeScope; label: string; hint: string }[] = [
  { value: "local", label: "local", hint: "this project, only you" },
  { value: "user", label: "user", hint: "all your projects" },
  { value: "project", label: "project", hint: "shared with the team via .mcp.json" },
];

function CodeBlock({ text }: { text: string }) {
  return (
    <div className="flex items-start justify-between gap-2 rounded-md border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2">
      <pre className="min-w-0 flex-1 overflow-x-auto whitespace-pre-wrap break-all font-mono text-[13px] text-[var(--color-text)]">
        {text}
      </pre>
      <CopyButton text={text} size="sm" />
    </div>
  );
}

export default function IntegrationsPage() {
  const [mcpUrl, setMcpUrl] = useState<string | null>(null);
  const [health, setHealth] = useState<HealthState>("checking");
  const [scope, setScope] = useState<ClaudeCodeScope>("user");

  // Hydration-safe: `window.location.origin` is only read after mount.
  useEffect(() => {
    setMcpUrl(mcpEndpointUrl(getConfig().apiUrl, window.location.origin));
  }, []);

  useEffect(() => {
    let cancelled = false;
    checkHealth()
      .then((ok) => {
        if (!cancelled) setHealth(ok ? "reachable" : "unreachable");
      })
      .catch(() => {
        if (!cancelled) setHealth("unreachable");
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const local = mcpUrl ? isLocalUrl(mcpUrl) : false;
  const selectedScope = SCOPE_OPTIONS.find((o) => o.value === scope);

  return (
    <div className="flex h-full flex-col">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.1, ease: "easeOut" }}
        className="flex-1 overflow-y-auto p-6 space-y-6"
      >
        <p className="max-w-3xl text-sm text-[var(--color-text-secondary)]">
          Control komputer from Claude Code, Claude Desktop, or any MCP-aware agent. Every
          API operation — creating agents, sending tasks, managing schedules, memories,
          skills, secrets and connectors — is available as an MCP tool.
        </p>

        {/* Step 1 — Your MCP endpoint */}
        <Card>
          <CardHeader>
            <CardTitle>Step 1 — Your MCP endpoint</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center gap-2">
              <div className="min-w-0 flex-1">
                {mcpUrl ? (
                  <CodeBlock text={mcpUrl} />
                ) : (
                  <div className="h-[38px] rounded-md border border-[var(--color-border)] bg-[var(--color-bg)]" />
                )}
              </div>
              <Badge
                variant="outline"
                className={
                  health === "reachable"
                    ? "border-green-500/30 bg-green-500/10 text-green-400"
                    : health === "unreachable"
                      ? "border-red-400/30 bg-red-400/10 text-red-400"
                      : undefined
                }
              >
                {health === "checking"
                  ? "Checking…"
                  : health === "reachable"
                    ? "Reachable"
                    : "Unreachable"}
              </Badge>
            </div>

            {local && (
              <p className="text-xs text-[var(--color-text-muted)]">
                This URL only works from this machine. From elsewhere, use your API&apos;s
                external address, or run{" "}
                <code className="font-mono">kubectl port-forward svc/komputer-api 8080:8080</code>.
              </p>
            )}

            <div className="flex items-start gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3">
              <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-400" />
              <p className="text-xs text-amber-300">
                No authentication: anyone who can reach this endpoint can create and delete
                agents and manage secrets. Keep it on an internal network.
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Step 2 — Connect a client */}
        <Card>
          <CardHeader>
            <CardTitle>Step 2 — Connect a client</CardTitle>
          </CardHeader>
          <CardContent>
            <Tabs defaultValue="claude-code">
              <TabsList>
                <TabsTrigger value="claude-code">Claude Code</TabsTrigger>
                <TabsTrigger value="claude-desktop">Claude Desktop</TabsTrigger>
                <TabsTrigger value="other">Other clients</TabsTrigger>
              </TabsList>

              <TabsContent value="claude-code" className="space-y-3 pt-4">
                <div className="flex flex-wrap items-center gap-1.5">
                  {SCOPE_OPTIONS.map((opt) => (
                    <button
                      key={opt.value}
                      type="button"
                      onClick={() => setScope(opt.value)}
                      className={`rounded-full border px-2.5 py-1 text-xs transition-colors cursor-pointer ${
                        scope === opt.value
                          ? "border-[var(--color-brand-blue)] bg-[var(--color-brand-blue)]/10 text-[var(--color-brand-blue)]"
                          : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
                      }`}
                    >
                      {opt.label}
                    </button>
                  ))}
                </div>
                {selectedScope && (
                  <p className="text-xs text-[var(--color-text-muted)]">
                    {selectedScope.value} = {selectedScope.hint}
                  </p>
                )}

                {mcpUrl && <CodeBlock text={claudeCodeAddCommand(mcpUrl, scope)} />}

                <div className="space-y-1">
                  <p className="text-xs text-[var(--color-text-secondary)]">Verify:</p>
                  <CodeBlock text="claude mcp list" />
                </div>

                <p className="text-xs text-[var(--color-text-muted)]">
                  Tools show up as <code className="font-mono">mcp__komputer__&lt;tool&gt;</code> —
                  try asking Claude to list your agents.
                </p>
              </TabsContent>

              <TabsContent value="claude-desktop" className="space-y-3 pt-4">
                <ol className="list-inside list-decimal space-y-2 text-sm text-[var(--color-text-secondary)]">
                  <li>Open Settings → Developer → Edit Config.</li>
                  <li>
                    Add this to <code className="font-mono">claude_desktop_config.json</code>{" "}
                    (merge into existing <code className="font-mono">mcpServers</code> if present):
                    {mcpUrl && (
                      <div className="mt-2">
                        <CodeBlock text={claudeDesktopConfig(mcpUrl)} />
                      </div>
                    )}
                  </li>
                  <li>Restart Claude Desktop.</li>
                </ol>
                <p className="text-xs text-[var(--color-text-muted)]">
                  Uses the <code className="font-mono">mcp-remote</code> bridge (requires
                  Node.js), so the connection runs from your machine and works with internal
                  URLs.
                </p>
              </TabsContent>

              <TabsContent value="other" className="space-y-2 pt-4">
                <p className="text-sm text-[var(--color-text-secondary)]">
                  Point any streamable-HTTP MCP client at the endpoint above.
                </p>
                <a
                  href={MCP_DOCS_URL}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 text-sm text-[var(--color-brand-blue)] hover:underline"
                >
                  MCP server docs
                  <ExternalLink className="size-3" />
                </a>
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>

        {/* Step 3 — Try it */}
        <Card>
          <CardHeader>
            <CardTitle>Step 3 — Try it</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {EXAMPLE_PROMPTS.map((prompt) => (
              <CodeBlock key={prompt} text={prompt} />
            ))}
          </CardContent>
        </Card>
      </motion.div>
    </div>
  );
}
