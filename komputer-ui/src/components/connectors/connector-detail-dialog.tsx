"use client";

/* eslint-disable @next/next/no-img-element */
import { useCallback, useEffect, useState } from "react";
import { motion } from "framer-motion";
import { Plug, Users, ExternalLink, Wrench, Calendar, Loader2, AlertCircle, KeyRound, Check } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/kit/dialog";
import { Button } from "@/components/kit/button";
import { Input } from "@/components/kit/input";
import { formatRelativeTime } from "@/lib/utils";
import { useConnectorTemplates } from "@/hooks/use-connector-templates";
import { ConnectorLogo } from "@/components/connectors/connector-logo";
import { getConnectorTools, updateConnector } from "@/lib/api";
import type { ConnectorResponse } from "@/lib/types";

type MCPTool = { name: string; description: string };

function groupToolsByPrefix(tools: MCPTool[]): { prefix: string; tools: MCPTool[] }[] {
  const sorted = [...tools].sort((a, b) => a.name.localeCompare(b.name));
  const groups = new Map<string, MCPTool[]>();
  for (const tool of sorted) {
    const prefix = tool.name.includes("_") ? tool.name.split("_")[0] : "other";
    if (!groups.has(prefix)) groups.set(prefix, []);
    groups.get(prefix)!.push(tool);
  }
  return Array.from(groups.entries()).map(([prefix, tools]) => ({ prefix, tools }));
}

type Props = {
  connector: ConnectorResponse | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpdated?: () => void;
};

export function ConnectorDetailDialog({ connector, open, onOpenChange, onUpdated }: Props) {
  const { getByService } = useConnectorTemplates();
  const [tools, setTools] = useState<MCPTool[]>([]);
  const [loading, setLoading] = useState(false);
  const [toolError, setToolError] = useState<string | null>(null);
  const [newToken, setNewToken] = useState("");
  const [savingToken, setSavingToken] = useState(false);
  const [tokenSaved, setTokenSaved] = useState(false);
  const [tokenError, setTokenError] = useState<string | null>(null);

  const loadTools = useCallback(() => {
    if (!connector) return;
    setTools([]);
    setToolError(null);
    setLoading(true);
    getConnectorTools(connector.name, connector.namespace)
      .then((res) => setTools(res.tools ?? []))
      .catch((e) => setToolError(e instanceof Error ? e.message : "Failed to fetch tools"))
      .finally(() => setLoading(false));
  }, [connector]);

  useEffect(() => {
    if (!open) return;
    setNewToken("");
    setTokenSaved(false);
    setTokenError(null);
    loadTools();
  }, [open, loadTools]);

  async function handleUpdateToken(e: React.FormEvent) {
    e.preventDefault();
    if (!connector || !newToken.trim()) return;
    setSavingToken(true);
    setTokenError(null);
    setTokenSaved(false);
    try {
      await updateConnector(connector.name, { token: newToken.trim(), namespace: connector.namespace });
      setNewToken("");
      setTokenSaved(true);
      onUpdated?.();
      // Re-fetch tools so the user sees right away whether the new token works.
      loadTools();
    } catch (err: unknown) {
      setTokenError(err instanceof Error ? err.message : "Failed to update token.");
    } finally {
      setSavingToken(false);
    }
  }

  if (!connector) return null;
  const canUpdateToken = connector.authType !== "oauth";

  const tpl = getByService(connector.service);
  const color = tpl?.color ?? "#8899A6";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl overflow-visible">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            <div className="flex items-center justify-center w-8 h-8 rounded-lg shrink-0">
              {tpl && tpl.logoUrl
                ? <ConnectorLogo src={tpl.logoUrl} alt={tpl.displayName} className="w-5 h-5" />
                : <Plug className="w-4 h-4" style={{ color }} />
              }
            </div>
            <div className="min-w-0">
              <span className="text-base font-semibold text-[var(--color-text)]">{connector.name}</span>
              <span className="ml-2 text-sm font-normal text-[var(--color-text-muted)]">{tpl?.displayName ?? connector.service}</span>
            </div>
          </DialogTitle>
        </DialogHeader>

        <div className="mt-5 space-y-5">
          {/* Meta row */}
          <div className="flex flex-wrap gap-4 text-[12px] text-[var(--color-text-secondary)]">
            {connector.url && (
              <span className="flex items-center gap-1.5">
                <ExternalLink className="w-3 h-3 text-[var(--color-text-muted)]" />
                {connector.url.replace(/^https?:\/\//, "")}
              </span>
            )}
            <span className="flex items-center gap-1.5">
              <Users className="w-3 h-3 text-[var(--color-text-muted)]" />
              {connector.attachedAgents} agent{connector.attachedAgents !== 1 ? "s" : ""}
            </span>
            <span className="flex items-center gap-1.5">
              <Calendar className="w-3 h-3 text-[var(--color-text-muted)]" />
              Created {formatRelativeTime(connector.createdAt)}
            </span>
          </div>

          {/* Attached agents */}
          {connector.agentNames && connector.agentNames.length > 0 && (
            <div>
              <p className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)] mb-2">Attached Agents</p>
              <div className="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto pr-1">
                {connector.agentNames.map((a) => (
                  <span key={a} className="text-[11px] px-2 py-0.5 rounded-full border border-[var(--color-border)] bg-[var(--color-surface-raised)] text-[var(--color-text-secondary)]">
                    {a}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Update auth token — OAuth connectors are reconnected via the OAuth flow instead */}
          {canUpdateToken && (
            <form onSubmit={handleUpdateToken}>
              <p className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)] mb-2 flex items-center gap-1.5">
                <KeyRound className="w-3 h-3" />
                Auth Token
              </p>
              <div className="flex items-center gap-2">
                <Input
                  type="password"
                  placeholder={connector.authSecretName ? "Paste a new token to replace the current one" : "Paste a token to add authentication"}
                  value={newToken}
                  onChange={(e) => { setNewToken(e.target.value); setTokenSaved(false); }}
                  autoComplete="off"
                  className="font-[family-name:var(--font-mono)] h-8 text-[12px]"
                />
                <Button type="submit" size="sm" disabled={savingToken || !newToken.trim()}>
                  {savingToken ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : "Update"}
                </Button>
              </div>
              {tokenSaved && (
                <p className="mt-1.5 flex items-center gap-1 text-[11px] text-green-400">
                  <Check className="w-3 h-3" /> Token updated. Running agents pick it up on their next start.
                </p>
              )}
              {tokenError && <p className="mt-1.5 text-[11px] text-red-400">{tokenError}</p>}
            </form>
          )}

          {/* Available tools */}
          <div>
            <p className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)] mb-2.5 flex items-center gap-1.5">
              <Wrench className="w-3 h-3" />
              Available Tools
              {!loading && tools.length > 0 && (
                <span className="font-normal text-[var(--color-text-muted)] normal-case tracking-normal">({tools.length})</span>
              )}
            </p>
            {loading ? (
              <div className="flex items-center gap-2 text-[12px] text-[var(--color-text-muted)]">
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
                Fetching tools…
              </div>
            ) : tools.length > 0 ? (
              <div className="space-y-4 max-h-64 overflow-y-auto pr-1">
                {groupToolsByPrefix(tools).map(({ prefix, tools: group }, gi) => (
                  <div key={prefix}>
                    <p className="text-[10px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)] mb-1.5 px-0.5">{prefix}</p>
                    <div className="space-y-1">
                      {group.map((tool, i) => (
                        <motion.div
                          key={tool.name}
                          className="flex items-start gap-3 rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2"
                          initial={{ opacity: 0, y: 6 }}
                          animate={{ opacity: 1, y: 0 }}
                          transition={{ duration: 0.15, delay: (gi * 3 + i) * 0.02 }}
                        >
                          <code className="text-[11px] font-[family-name:var(--font-mono)] text-[var(--color-brand-blue)] shrink-0 mt-0.5">{tool.name}</code>
                          <span className="text-[11px] text-[var(--color-text-secondary)]">{tool.description}</span>
                        </motion.div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            ) : toolError ? (
              (() => {
                const urlMatch = toolError.match(/https?:\/\/\S+/);
                const url = urlMatch?.[0];
                const msg = url ? toolError.replace(url, "").trim().replace(/\.*$/, "") : toolError;
                return (
                  <div className="flex flex-col gap-2">
                    <div className="flex items-start gap-2 text-[12px] text-[var(--color-text-muted)]">
                      <AlertCircle className="w-3.5 h-3.5 shrink-0 mt-0.5 text-amber-400" />
                      <span>{msg}</span>
                    </div>
                    {url && (
                      <a href={url} target="_blank" rel="noopener noreferrer" className="ml-5 text-[12px] text-[var(--color-brand-blue-light)] hover:underline break-all">
                        {url}
                      </a>
                    )}
                  </div>
                );
              })()
            ) : (
              <p className="text-[12px] text-[var(--color-text-muted)]">No tools found — the MCP server may be unreachable or require authentication.</p>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
