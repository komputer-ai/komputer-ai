"use client";
/* eslint-disable @next/next/no-img-element */

import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/kit/dialog";
import { Button } from "@/components/kit/button";
import { Input } from "@/components/kit/input";
import { Label } from "@/components/kit/label";
import { NamespaceSelector } from "@/components/shared/namespace-selector";
import { createConnector, createSecretResource, getOAuthAuthorizeUrl } from "@/lib/api";
import { useConnectorTemplates } from "@/hooks/use-connector-templates";
import type { ConnectorTemplate } from "@/lib/types";
import { ArrowLeft, Copy, Check, Plug } from "lucide-react";

const NAME_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/;

// Turns URLs and domain-like strings (e.g. github.com/settings) into clickable links.
function linkify(text: string) {
  const urlRegex = /(https?:\/\/[^\s,)]+|[a-z0-9-]+\.[a-z]{2,}[^\s,)]*)/gi;
  const parts = text.split(urlRegex);
  if (parts.length === 1) return text;
  return parts.map((part, i) =>
    urlRegex.test(part) ? (
      <a key={i} href={part.startsWith("http") ? part : `https://${part}`} target="_blank" rel="noopener noreferrer" className="text-[var(--color-brand-blue-light)] hover:underline">{part}</a>
    ) : (
      <span key={i}>{part}</span>
    )
  );
}

function ManifestBlock({ manifest }: { manifest: string }) {
  const [copied, setCopied] = useState(false);
  function handleCopy() {
    navigator.clipboard.writeText(manifest);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }
  return (
    <div className="mt-1 rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-[var(--color-border)]">
        <span className="text-[11px] font-medium text-[var(--color-text-muted)]">App Manifest</span>
        <button
          type="button"
          onClick={handleCopy}
          className="flex items-center gap-1 text-[11px] text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors cursor-pointer"
        >
          {copied ? <Check className="w-3 h-3 text-green-400" /> : <Copy className="w-3 h-3" />}
          {copied ? "Copied!" : "Copy"}
        </button>
      </div>
      <pre className="px-3 py-2.5 text-[10px] font-mono text-[var(--color-text-secondary)] max-h-36 overflow-y-auto leading-relaxed whitespace-pre-wrap break-all">{manifest}</pre>
    </div>
  );
}

type CreateConnectorModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: () => void;
  initialTemplate?: ConnectorTemplate;
};

export function CreateConnectorModal({ open, onOpenChange, onCreated, initialTemplate }: CreateConnectorModalProps) {
  const { templates, loading: templatesLoading } = useConnectorTemplates();
  const [step, setStep] = useState<"pick" | "form">(initialTemplate ? "form" : "pick");
  const [selectedTemplate, setSelectedTemplate] = useState<ConnectorTemplate | null>(initialTemplate ?? null);
  const [name, setName] = useState("");
  const [namespace, setNamespace] = useState("default");
  const [url, setUrl] = useState("");
  const [credential, setCredential] = useState("");
  const [oauthClientId, setOauthClientId] = useState("");
  const [oauthClientSecret, setOauthClientSecret] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isCustom = selectedTemplate?.service === "custom";
  const isOAuth = selectedTemplate?.authType === "oauth";
  const isNoAuth = selectedTemplate?.authType === "none";
  const isKnownTemplate = selectedTemplate && selectedTemplate.url && !isCustom;

  useEffect(() => {
    if (initialTemplate && open) {
      handlePickTemplate(initialTemplate);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialTemplate, open]);

  function resetForm() {
    setStep("pick");
    setSelectedTemplate(null);
    setName("");
    setNamespace("default");
    setUrl("");
    setCredential("");
    setOauthClientId("");
    setOauthClientSecret("");
    setError(null);
  }

  function handlePickTemplate(tpl: ConnectorTemplate) {
    setSelectedTemplate(tpl);
    setName(tpl.service === "custom" ? "" : tpl.service);
    setUrl(tpl.service === "custom" ? "" : tpl.url);
    setCredential("");
    setStep("form");
  }

  function validate(): string | null {
    if (!name.trim()) return "Name is required.";
    if (!NAME_PATTERN.test(name)) return "Name must be lowercase letters, numbers, and hyphens only.";
    if (!url.trim()) return "URL is required.";
    if (isKnownTemplate && !isNoAuth && !credential.trim()) return `${selectedTemplate.authLabel} is required.`;
    return null;
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const validationError = validate();
    if (validationError) {
      setError(validationError);
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      let secretName: string | undefined;
      let secretKey: string | undefined;

      // Auto-create a K8s secret for both known templates and custom connectors
      if (credential.trim()) {
        const autoSecretName = `${name.trim()}-credentials`;
        const dataKey = "token";
        await createSecretResource({
          name: autoSecretName,
          data: { [dataKey]: credential.trim() },
          namespace: namespace.trim() || undefined,
        });
        secretName = autoSecretName;
        secretKey = dataKey;
      }

      await createConnector({
        name: name.trim(),
        service: selectedTemplate?.service ?? name.trim(),
        displayName: selectedTemplate?.displayName,
        url: url.trim(),
        authSecretName: secretName,
        authSecretKey: secretKey,
        namespace: namespace.trim() || undefined,
      });
      resetForm();
      onOpenChange(false);
      onCreated?.();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to create connector.");
    } finally {
      setSubmitting(false);
    }
  }

  function validateOAuth(): string | null {
    if (!name.trim()) return "Name is required.";
    if (!NAME_PATTERN.test(name)) return "Name must be lowercase letters, numbers, and hyphens only.";
    return null;
  }

  async function handleOAuthSubmit(e: React.FormEvent) {
    e.preventDefault();
    const validationError = validateOAuth();
    if (validationError) { setError(validationError); return; }

    setSubmitting(true);
    setError(null);

    try {
      // Get OAuth authorize URL — connector is created only after successful callback.
      const { authorizeUrl } = await getOAuthAuthorizeUrl({
        service: selectedTemplate!.service,
        connector_name: name.trim(),
        displayName: selectedTemplate!.displayName,
        url: selectedTemplate!.url,
        oauthClientId: oauthClientId.trim(),
        oauthClientSecret: oauthClientSecret.trim(),
        namespace: namespace.trim() || undefined,
      });

      // 3. Open popup
      const popup = window.open(authorizeUrl, "oauth-popup", "width=600,height=700,scrollbars=yes");

      // 4. Listen for success — postMessage (same-origin) + localStorage (cross-origin fallback)
      let resolved = false;
      const onSuccess = () => {
        if (resolved) return;
        resolved = true;
        cleanup();
        resetForm();
        onOpenChange(false);
        onCreated?.();
      };

      const handleMessage = (event: MessageEvent) => {
        if (event.data?.type === "oauth-success") onSuccess();
      };
      window.addEventListener("message", handleMessage);

      // localStorage fallback: the callback page writes "oauth-success" key
      const handleStorage = (event: StorageEvent) => {
        if (event.key === "oauth-success") {
          localStorage.removeItem("oauth-success");
          onSuccess();
        }
      };
      window.addEventListener("storage", handleStorage);

      // 5. Poll for popup close (user cancelled)
      const checkClosed = setInterval(() => {
        if (popup?.closed) {
          // Give a short grace period for the success signal to arrive
          setTimeout(() => {
            if (!resolved) {
              cleanup();
              setSubmitting(false);
            }
          }, 1000);
        }
      }, 500);

      const cleanup = () => {
        clearInterval(checkClosed);
        window.removeEventListener("message", handleMessage);
        window.removeEventListener("storage", handleStorage);
      };

    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to start OAuth flow.");
      setSubmitting(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        onOpenChange(nextOpen);
        if (!nextOpen) resetForm();
      }}
    >
      <DialogContent className="max-w-3xl max-h-[85vh] flex flex-col">
        <AnimatePresence mode="wait">
          {step === "pick" ? (
            <motion.div
              key="pick"
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -20 }}
              transition={{ duration: 0.2, ease: "easeOut" }}
              className="flex flex-col min-h-0 flex-1"
            >
              <DialogHeader>
                <DialogTitle className="text-lg">Add Connector</DialogTitle>
                <DialogDescription className="text-sm">
                  Choose a service to connect your agents to.
                </DialogDescription>
              </DialogHeader>
              <div className="mt-6 grid grid-cols-2 sm:grid-cols-3 gap-3 overflow-y-auto">
                {templatesLoading ? (
                  <div className="col-span-3 flex items-center justify-center py-12 text-sm text-[var(--color-text-muted)]">Loading...</div>
                ) : templates.map((tpl, i) => (
                  <motion.button
                    key={tpl.service}
                    type="button"
                    disabled={!tpl.url}
                    onClick={() => handlePickTemplate(tpl)}
                    initial={{ opacity: 0, y: 12, scale: 0.97 }}
                    animate={{ opacity: 1, y: 0, scale: 1 }}
                    transition={{ duration: 0.2, delay: i * 0.04 }}
                    className="group relative flex flex-col items-center gap-3 rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-6 transition-all duration-200 hover:border-[var(--color-border-hover)] hover:shadow-[0_0_24px_rgba(139,92,246,0.08)] cursor-pointer text-center disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:border-[var(--color-border)]"
                  >
                    <div className="flex items-center justify-center w-12 h-12 rounded-xl transition-transform duration-200 group-hover:scale-110">
                      {tpl.service === "custom"
                        ? <Plug className="w-7 h-7 text-[var(--color-text-secondary)]" />
                        : <img src={tpl.logoUrl} alt={tpl.displayName} className="w-7 h-7" />
                      }
                    </div>
                    <div>
                      <p className="text-sm font-semibold text-[var(--color-text)]">{tpl.displayName}</p>
                      <p className="mt-1 text-[11px] text-[var(--color-text-secondary)] line-clamp-2">{tpl.description}</p>
                    </div>
                    {!tpl.url && (
                      <span className="absolute top-2 right-2 text-[8px] tracking-wider uppercase px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-400 leading-none">
                        soon
                      </span>
                    )}
                  </motion.button>
                ))}
              </div>
            </motion.div>
          ) : (
            <motion.div
              key="form"
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: 20 }}
              transition={{ duration: 0.2, ease: "easeOut" }}
              className="flex flex-col min-h-0 flex-1"
            >
              <form onSubmit={isOAuth ? handleOAuthSubmit : handleSubmit} className="flex flex-col min-h-0 flex-1">
                <DialogHeader>
                  <DialogTitle className="flex items-center gap-3 text-lg">
                    <button type="button" onClick={() => setStep("pick")} className="p-1.5 -ml-1 rounded-md hover:bg-[var(--color-surface-hover)] transition-colors cursor-pointer">
                      <ArrowLeft className="size-4 text-[var(--color-text-secondary)]" />
                    </button>
                    {selectedTemplate && (isCustom
                      ? <Plug className="w-5 h-5 text-[var(--color-text-secondary)]" />
                      : <img src={selectedTemplate.logoUrl} alt={selectedTemplate.displayName} className="w-6 h-6" />
                    )}
                    {isCustom ? "Custom Connector" : `Connect ${selectedTemplate?.displayName}`}
                  </DialogTitle>
                </DialogHeader>

                <div className="mt-6 flex flex-col gap-5 min-h-0 flex-1">
                  {/* Step-by-step guide — scrollable if tall */}
                  {isKnownTemplate && selectedTemplate.guideSteps.length > 1 && (
                    <motion.div
                      className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg)] overflow-y-auto min-h-0 shrink"
                      style={{ maxHeight: "40vh" }}
                      initial={{ opacity: 0, y: 8 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ duration: 0.25, delay: 0.1 }}
                    >
                      <div className="px-5 py-3 border-b border-[var(--color-border)] sticky top-0 bg-[var(--color-bg)] z-10">
                        <p className="text-sm font-medium text-[var(--color-text-secondary)]">How to get your {selectedTemplate.authLabel.toLowerCase()}</p>
                      </div>
                      <div className="px-5 py-4 space-y-3.5">
                        {selectedTemplate.guideSteps.map((guideStep, i) => (
                          <>
                          <motion.div
                            key={i}
                            className="flex items-start gap-3"
                            initial={{ opacity: 0, x: -8 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ duration: 0.2, delay: 0.15 + i * 0.06 }}
                          >
                            <span className="flex items-center justify-center w-6 h-6 rounded-full text-[11px] font-semibold shrink-0 mt-0.5" style={{ backgroundColor: `${selectedTemplate.color}20`, color: selectedTemplate.color }}>
                              {i + 1}
                            </span>
                            <p className="text-sm text-[var(--color-text-secondary)] leading-relaxed pt-0.5">{linkify(guideStep)}</p>
                          </motion.div>
                          {selectedTemplate.manifest && selectedTemplate.manifestAfterStep === i && (
                            <ManifestBlock manifest={selectedTemplate.manifest} />
                          )}
                          </>
                        ))}
                      </div>
                    </motion.div>
                  )}

                  {/* Form fields — never clipped, dropdowns work */}
                  <div className="flex flex-col gap-5 shrink-0">
                    <div className="flex flex-col gap-2">
                      <Label htmlFor="conn-name">Name</Label>
                      <Input
                        id="conn-name"
                        placeholder="company-github"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        autoComplete="off"
                      />
                    </div>

                    <NamespaceSelector value={namespace} onChange={setNamespace} />

                    {isCustom && (
                      <div className="flex flex-col gap-2">
                        <Label htmlFor="conn-url">MCP Server URL</Label>
                        <Input
                          id="conn-url"
                          placeholder="https://mcp.example.com"
                          value={url}
                          onChange={(e) => setUrl(e.target.value)}
                          autoComplete="off"
                        />
                      </div>
                    )}

                    {isOAuth && oauthClientId && (
                      <>
                        <div className="flex flex-col gap-2">
                          <Label htmlFor="conn-client-id">Client ID</Label>
                          <Input
                            id="conn-client-id"
                            placeholder="OAuth Client ID (optional — auto-registered if supported)"
                            value={oauthClientId}
                            onChange={(e) => setOauthClientId(e.target.value)}
                            autoComplete="off"
                            className="font-[family-name:var(--font-mono)]"
                          />
                        </div>
                        <div className="flex flex-col gap-2">
                          <Label htmlFor="conn-client-secret">Client Secret</Label>
                          <Input
                            id="conn-client-secret"
                            type="password"
                            placeholder="OAuth Client Secret"
                            value={oauthClientSecret}
                            onChange={(e) => setOauthClientSecret(e.target.value)}
                            autoComplete="off"
                            className="font-[family-name:var(--font-mono)]"
                          />
                        </div>
                      </>
                    )}

                    {!isOAuth && !isNoAuth && (
                      <div className="flex flex-col gap-2">
                        <Label htmlFor="conn-cred">{isKnownTemplate ? selectedTemplate.authLabel : "Auth Token"}</Label>
                        <Input
                          id="conn-cred"
                          type="password"
                          placeholder={isKnownTemplate ? selectedTemplate.authPlaceholder : "optional"}
                          value={credential}
                          onChange={(e) => setCredential(e.target.value)}
                          autoComplete="off"
                          className="font-[family-name:var(--font-mono)]"
                        />
                      </div>
                    )}

                    {error && (
                      <p className="text-sm text-red-400">{error}</p>
                    )}
                  </div>
                </div>

                <DialogFooter className="mt-6 shrink-0">
                  <Button variant="secondary" type="button" onClick={() => onOpenChange(false)}>
                    Cancel
                  </Button>
                  <Button type="submit" disabled={submitting}>
                    {isOAuth ? `Connect with ${selectedTemplate?.displayName}` : submitting ? "Connecting..." : "Connect"}
                  </Button>
                </DialogFooter>
              </form>
            </motion.div>
          )}
        </AnimatePresence>
      </DialogContent>
    </Dialog>
  );
}
