"use client";

import { useEffect, useRef, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { Input } from "@/components/kit/input";
import { Textarea } from "@/components/kit/textarea";
import { Label } from "@/components/kit/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/kit/select";
import { MultiSelect, type MultiSelectOption } from "@/components/kit/multi-select";
import { ChevronRight, Plus } from "lucide-react";
import { CreateSecretModal } from "@/components/secrets/create-secret-modal";
import { ConnectorLogo } from "@/components/connectors/connector-logo";
import { useConnectorTemplates } from "@/hooks/use-connector-templates";
import { NamespaceSelector } from "@/components/shared/namespace-selector";
import { listTemplates, listMemories, listSkills, listSecrets, listConnectors } from "@/lib/api";
import type { TemplateResponse } from "@/lib/types";
import { MODELS, LIFECYCLES } from "@/lib/constants";

export interface AgentFormValues {
  name: string;
  namespace: string;
  instructions: string;
  systemPrompt: string;
  model: string;
  lifecycle: string;
  role: "manager" | "worker" | undefined;
  templateRef: string;
  selectedMemories: string[];
  selectedSkills: string[];
  selectedConnectors: string[];
  selectedSecretRefs: string[];
  priority: number;
  cpu: string;
  memoryLimit: string;
  storageSize: string;
  image: string;
  // UI-only state (preserved across tab switches)
  systemPromptOpen: boolean;
  advancedOpen: boolean;
  showAllSecrets: boolean;
}

export function makeDefaultAgentFormValues(overrides?: Partial<AgentFormValues>): AgentFormValues {
  return {
    name: "",
    namespace: "default",
    instructions: "",
    systemPrompt: "",
    model: "claude-sonnet-4-6",
    lifecycle: "default",
    role: undefined,
    templateRef: "default",
    selectedMemories: [],
    selectedSkills: [],
    selectedConnectors: [],
    selectedSecretRefs: [],
    priority: 0,
    cpu: "",
    memoryLimit: "",
    storageSize: "",
    image: "",
    systemPromptOpen: false,
    advancedOpen: false,
    showAllSecrets: false,
    ...overrides,
  };
}

export interface AgentFieldsFormProps {
  values: AgentFormValues;
  onChange: (values: AgentFormValues) => void;
  /** Whether this form is currently visible (controls when to fetch options) */
  active: boolean;
  /** When true, hide both Name and Namespace fields */
  hideNameAndNamespace?: boolean;
  /** When true, hide only the Namespace field (squad mode — squad owns namespace, agent still gets a name) */
  hideNamespaceOnly?: boolean;
  /** Optional id prefix for input ids (avoid collisions when multiple forms exist) */
  idPrefix?: string;
}

export function AgentFieldsForm({
  values,
  onChange,
  active,
  hideNameAndNamespace = false,
  hideNamespaceOnly = false,
  idPrefix = "agent",
}: AgentFieldsFormProps) {
  const advancedRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  const { getByService: getConnectorTemplate } = useConnectorTemplates();

  const [templates, setTemplates] = useState<TemplateResponse[]>([]);
  const [availableMemories, setAvailableMemories] = useState<{ name: string; namespace: string; ref: string }[]>([]);
  const [availableSkills, setAvailableSkills] = useState<{ name: string; namespace: string; ref: string }[]>([]);
  const [availableConnectors, setAvailableConnectors] = useState<{ name: string; namespace: string; ref: string; service: string }[]>([]);
  const [availableSecrets, setAvailableSecrets] = useState<{ name: string; namespace: string }[]>([]);
  const [createSecretOpen, setCreateSecretOpen] = useState(false);

  const { namespace, showAllSecrets } = values;

  useEffect(() => {
    if (!active) return;
    listTemplates(namespace || undefined)
      .then((res) => setTemplates(res.templates ?? []))
      .catch(() => setTemplates([]));
    listMemories()
      .then((res) => setAvailableMemories((res.memories ?? []).map((m) => ({
        name: m.name,
        namespace: m.namespace,
        ref: m.namespace === (namespace || "default") ? m.name : `${m.namespace}/${m.name}`,
      }))))
      .catch(() => setAvailableMemories([]));
    listSkills()
      .then((res) => setAvailableSkills((res.skills ?? []).map((s) => ({
        name: s.name,
        namespace: s.namespace,
        ref: s.namespace === (namespace || "default") ? s.name : `${s.namespace}/${s.name}`,
      }))))
      .catch(() => setAvailableSkills([]));
    listSecrets(namespace || undefined, showAllSecrets)
      .then((res) => setAvailableSecrets((res.secrets ?? []).map((s) => ({ name: s.name, namespace: s.namespace }))))
      .catch(() => setAvailableSecrets([]));
    listConnectors()
      .then((res) => setAvailableConnectors((res.connectors ?? []).map((c) => ({
        name: c.name,
        namespace: c.namespace,
        ref: c.namespace === (namespace || "default") ? c.name : `${c.namespace}/${c.name}`,
        service: c.service,
      }))))
      .catch(() => setAvailableConnectors([]));
  }, [active, namespace, showAllSecrets]);

  function patch<K extends keyof AgentFormValues>(key: K, value: AgentFormValues[K]) {
    onChange({ ...values, [key]: value });
  }

  return (
    <div ref={scrollRef} className="flex flex-col gap-4">
      {!hideNameAndNamespace && (
        hideNamespaceOnly ? (
          <div className="flex flex-col gap-1.5">
            <Label htmlFor={`${idPrefix}-name`}>Agent Name</Label>
            <Input
              id={`${idPrefix}-name`}
              placeholder="my-agent"
              value={values.name}
              onChange={(e) => patch("name", e.target.value)}
              autoComplete="off"
            />
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor={`${idPrefix}-name`}>Name</Label>
              <Input
                id={`${idPrefix}-name`}
                placeholder="my-agent"
                value={values.name}
                onChange={(e) => patch("name", e.target.value)}
                autoComplete="off"
              />
            </div>
            <NamespaceSelector value={values.namespace} onChange={(v) => patch("namespace", v)} />
          </div>
        )
      )}

      {/* Collapsible system prompt */}
      <div className="flex flex-col">
        <button
          type="button"
          className="flex items-center gap-1.5 text-sm font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors cursor-pointer"
          onClick={() => patch("systemPromptOpen", !values.systemPromptOpen)}
        >
          <ChevronRight className={`size-3.5 transition-transform duration-150 ${values.systemPromptOpen ? "rotate-90" : ""}`} />
          System Prompt <span className="text-[var(--color-text-muted)] font-normal">(Optional)</span>
        </button>
        <AnimatePresence initial={false}>
          {values.systemPromptOpen && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: "auto", opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.15, ease: "easeOut" }}
              className="overflow-hidden"
            >
              <div className="pt-2">
                <Textarea
                  id={`${idPrefix}-system-prompt`}
                  placeholder="Custom instructions that define agent behavior, persona, or constraints..."
                  value={values.systemPrompt}
                  onChange={(e) => patch("systemPrompt", e.target.value)}
                  style={{ minHeight: 100 }}
                />
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor={`${idPrefix}-instructions`}>Instructions</Label>
        <Textarea
          id={`${idPrefix}-instructions`}
          placeholder="Describe what this agent should do..."
          value={values.instructions}
          onChange={(e) => patch("instructions", e.target.value)}
          style={{ minHeight: 200 }}
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="flex flex-col gap-1.5">
          <Label>Model</Label>
          <Select value={values.model} onValueChange={(v) => v && patch("model", v)}>
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {MODELS.map((m) => (
                <SelectItem key={m.value} value={m.value}>
                  {m.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Lifecycle</Label>
          <Select value={values.lifecycle} onValueChange={(v) => v && patch("lifecycle", v)}>
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {LIFECYCLES.map((l) => (
                <SelectItem key={l.value} value={l.value}>
                  {l.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label>Secrets</Label>
        <MultiSelect
          options={availableSecrets.map<MultiSelectOption>((s) => ({
            value: s.name,
            label: s.name,
            secondary: values.showAllSecrets ? s.namespace : null,
            searchTerms: [s.namespace],
          }))}
          value={values.selectedSecretRefs}
          onChange={(next) => patch("selectedSecretRefs", next)}
          placeholder="Select secrets..."
          noun="secrets"
          searchPlaceholder="Search secrets..."
          emptyText="No secrets available"
          headerExtra={
            <button
              type="button"
              onClick={() => patch("showAllSecrets", !values.showAllSecrets)}
              className={`w-full text-xs px-2 py-1 rounded border transition-colors cursor-pointer ${
                values.showAllSecrets
                  ? "border-amber-500/50 bg-amber-500/10 text-amber-400"
                  : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)]"
              }`}
            >
              {values.showAllSecrets ? "Showing all namespaces" : "Show all namespaces"}
            </button>
          }
          footerExtra={
            <button
              type="button"
              onClick={() => setCreateSecretOpen(true)}
              className="flex w-full items-center justify-center gap-1 text-xs px-2 py-1 rounded border border-dashed border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-hover)] hover:text-[var(--color-text)] transition-colors cursor-pointer"
            >
              <Plus className="size-3" />
              New Secret
            </button>
          }
        />
      </div>
      <CreateSecretModal
        open={createSecretOpen}
        onOpenChange={setCreateSecretOpen}
        onCreated={() => {
          listSecrets(values.namespace || undefined, values.showAllSecrets)
            .then((res) => setAvailableSecrets((res.secrets ?? []).map((s) => ({ name: s.name, namespace: s.namespace }))))
            .catch(() => {});
        }}
      />

      {/* Advanced section */}
      <div ref={advancedRef} className="rounded-md border border-[var(--color-border)]">
        <button
          type="button"
          onClick={() => patch("advancedOpen", !values.advancedOpen)}
          className="flex w-full items-center gap-2 px-3 py-2 text-left cursor-pointer hover:bg-[var(--color-surface-hover)] transition-colors"
        >
          <ChevronRight
            className={`size-3.5 shrink-0 text-[var(--color-text-secondary)] transition-transform duration-200 ${values.advancedOpen ? "rotate-90" : ""}`}
          />
          <span className="text-xs font-medium text-[var(--color-text-secondary)]">Advanced</span>
        </button>
        <AnimatePresence initial={false}>
          {values.advancedOpen && (
            <motion.div
              initial={{ height: 0, opacity: 0, overflow: "hidden" }}
              animate={{ height: "auto", opacity: 1, overflow: "visible", transitionEnd: { overflow: "visible" } }}
              exit={{ height: 0, opacity: 0, overflow: "hidden" }}
              transition={{ duration: 0.2, ease: "easeOut" }}
            >
              <div className="border-t border-[var(--color-border)] px-3 py-3 flex flex-col gap-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="flex flex-col gap-1.5">
                    <Label>Template</Label>
                    <Select value={values.templateRef} onValueChange={(v) => v && patch("templateRef", v)}>
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {templates.map((t) => (
                          <SelectItem key={`${t.scope}-${t.name}`} value={t.name}>
                            <span className="flex items-center gap-2">
                              {t.name}
                              <span className={`text-[10px] tracking-wider px-1.5 py-0.5 rounded ${t.scope === "cluster" ? "bg-[var(--color-brand-violet)]/10 text-[var(--color-brand-violet)]" : "bg-emerald-500/10 text-emerald-400"}`}>
                                {t.scope === "cluster" ? "cluster" : t.namespace}
                              </span>
                            </span>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  {/* Connectors */}
                  <div className="flex flex-col gap-1.5">
                    <Label>Connectors</Label>
                    <MultiSelect
                      options={availableConnectors.map<MultiSelectOption>((c) => {
                        const tpl = getConnectorTemplate(c.service);
                        return {
                          value: c.ref,
                          label: c.name,
                          secondary: c.ref.includes("/") ? c.namespace : null,
                          searchTerms: [c.namespace, c.ref, c.service],
                          icon: tpl?.logoUrl ? (
                            <ConnectorLogo src={tpl.logoUrl} alt={tpl.displayName} className="h-4 w-4" />
                          ) : undefined,
                        };
                      })}
                      value={values.selectedConnectors}
                      onChange={(next) => patch("selectedConnectors", next)}
                      placeholder="Select connectors..."
                      noun="connectors"
                      searchPlaceholder="Search connectors..."
                      emptyText="No connectors available"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  {/* Memories */}
                  <div className="flex flex-col gap-1.5">
                    <Label>Memories</Label>
                    <MultiSelect
                      options={availableMemories.map<MultiSelectOption>((m) => ({
                        value: m.ref,
                        label: m.name,
                        secondary: m.ref.includes("/") ? m.namespace : null,
                        searchTerms: [m.namespace, m.ref],
                      }))}
                      value={values.selectedMemories}
                      onChange={(next) => patch("selectedMemories", next)}
                      placeholder="Select memories..."
                      noun="memories"
                      searchPlaceholder="Search memories..."
                      emptyText="No memories available"
                    />
                  </div>

                  {/* Skills */}
                  <div className="flex flex-col gap-1.5">
                    <Label>Skills</Label>
                    <MultiSelect
                      options={availableSkills.map<MultiSelectOption>((s) => ({
                        value: s.ref,
                        label: s.name,
                        secondary: s.ref.includes("/") ? s.namespace : null,
                        searchTerms: [s.namespace, s.ref],
                      }))}
                      value={values.selectedSkills}
                      onChange={(next) => patch("selectedSkills", next)}
                      placeholder="Select skills..."
                      noun="skills"
                      searchPlaceholder="Search skills..."
                      emptyText="No skills available"
                    />
                  </div>
                </div>

                {/* Resource overrides */}
                <div className="pt-3 border-t border-[var(--color-border)] flex flex-col gap-3">
                  <span className="text-[10px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)]">
                    Template Overrides
                  </span>
                  <div className="grid grid-cols-2 gap-3">
                    <div className="flex flex-col gap-1.5">
                      <Label htmlFor={`${idPrefix}-cpu`}>CPU</Label>
                      <Input
                        id={`${idPrefix}-cpu`}
                        placeholder="e.g. 2 or 500m"
                        value={values.cpu}
                        onChange={(e) => patch("cpu", e.target.value)}
                        autoComplete="off"
                      />
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <Label htmlFor={`${idPrefix}-memory`}>Memory</Label>
                      <Input
                        id={`${idPrefix}-memory`}
                        placeholder="e.g. 4Gi"
                        value={values.memoryLimit}
                        onChange={(e) => patch("memoryLimit", e.target.value)}
                        autoComplete="off"
                      />
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <Label htmlFor={`${idPrefix}-storage`}>Storage</Label>
                      <Input
                        id={`${idPrefix}-storage`}
                        placeholder="e.g. 20Gi"
                        value={values.storageSize}
                        onChange={(e) => patch("storageSize", e.target.value)}
                        autoComplete="off"
                      />
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <Label htmlFor={`${idPrefix}-image`}>Container Image</Label>
                      <Input
                        id={`${idPrefix}-image`}
                        placeholder="e.g. custom:latest"
                        value={values.image}
                        onChange={(e) => patch("image", e.target.value)}
                        autoComplete="off"
                      />
                    </div>
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor={`${idPrefix}-priority`}>Priority</Label>
                    <input
                      id={`${idPrefix}-priority`}
                      type="number"
                      value={values.priority}
                      onChange={(e) => patch("priority", parseInt(e.target.value, 10) || 0)}
                      placeholder="0"
                      className="flex h-9 w-full rounded-md border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-1 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-muted)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--color-brand-blue)]"
                    />
                    <p className="text-xs text-[var(--color-text-secondary)]">Higher priority agents are admitted first when the template capacity limit is reached. Default: 0.</p>
                  </div>
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}

/** Build the CreateAgentRequest body from form values. */
export function buildCreateAgentRequest(values: AgentFormValues, opts?: { includeNamespace?: boolean }) {
  const includeNamespace = opts?.includeNamespace ?? true;
  let podSpecOverride: Record<string, unknown> | undefined;
  if (values.cpu.trim() || values.memoryLimit.trim() || values.image.trim()) {
    const container: Record<string, unknown> = { name: "agent" };
    if (values.image.trim()) container.image = values.image.trim();
    if (values.cpu.trim() || values.memoryLimit.trim()) {
      const rl: Record<string, string> = {};
      if (values.cpu.trim()) rl.cpu = values.cpu.trim();
      if (values.memoryLimit.trim()) rl.memory = values.memoryLimit.trim();
      container.resources = { requests: rl, limits: rl };
    }
    podSpecOverride = { containers: [container] };
  }

  return {
    name: values.name.trim(),
    instructions: values.instructions.trim(),
    model: values.model,
    namespace: includeNamespace ? (values.namespace.trim() || undefined) : undefined,
    lifecycle: values.lifecycle === "default" ? "" : (values.lifecycle as "" | "Sleep" | "AutoDelete"),
    role: values.role || undefined,
    templateRef: values.templateRef !== "default" ? values.templateRef : undefined,
    secretRefs: values.selectedSecretRefs.length > 0 ? values.selectedSecretRefs : undefined,
    memories: values.selectedMemories.length > 0 ? values.selectedMemories : undefined,
    skills: values.selectedSkills.length > 0 ? values.selectedSkills : undefined,
    connectors: values.selectedConnectors.length > 0 ? values.selectedConnectors : undefined,
    systemPrompt: values.systemPrompt.trim() || undefined,
    priority: values.priority !== 0 ? values.priority : undefined,
    podSpec: podSpecOverride,
    storage: values.storageSize.trim() ? { size: values.storageSize.trim() } : undefined,
  };
}

/** Build a KomputerAgentSpec (raw K8s field names) for embedding in a squad member.spec. */
export function buildAgentSpecForSquad(values: AgentFormValues): Record<string, unknown> {
  let podSpecOverride: Record<string, unknown> | undefined;
  if (values.cpu.trim() || values.memoryLimit.trim() || values.image.trim()) {
    const container: Record<string, unknown> = { name: "agent" };
    if (values.image.trim()) container.image = values.image.trim();
    if (values.cpu.trim() || values.memoryLimit.trim()) {
      const rl: Record<string, string> = {};
      if (values.cpu.trim()) rl.cpu = values.cpu.trim();
      if (values.memoryLimit.trim()) rl.memory = values.memoryLimit.trim();
      container.resources = { requests: rl, limits: rl };
    }
    podSpecOverride = { containers: [container] };
  }
  const spec: Record<string, unknown> = {
    instructions: values.instructions.trim(),
    model: values.model,
  };
  if (values.lifecycle && values.lifecycle !== "default") spec.lifecycle = values.lifecycle;
  if (values.role) spec.role = values.role;
  if (values.templateRef && values.templateRef !== "default") spec.templateRef = values.templateRef;
  if (values.systemPrompt.trim()) spec.systemPrompt = values.systemPrompt.trim();
  if (values.selectedSecretRefs.length > 0) spec.secrets = values.selectedSecretRefs;
  if (values.selectedMemories.length > 0) spec.memories = values.selectedMemories;
  if (values.selectedSkills.length > 0) spec.skills = values.selectedSkills;
  if (values.selectedConnectors.length > 0) spec.connectors = values.selectedConnectors;
  if (values.priority !== 0) spec.priority = values.priority;
  if (podSpecOverride) spec.podSpec = podSpecOverride;
  if (values.storageSize.trim()) spec.storage = { size: values.storageSize.trim() };
  return spec;
}

