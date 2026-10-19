import { describe, it, expect } from "vitest";
import {
  makeDefaultAgentFormValues,
  buildScheduleAgentSpec,
  agentFormValuesFromScheduleSpec,
  buildCreateAgentRequest,
  buildAgentSpecForSquad,
} from "./agent-fields-form";
import type { ScheduleAgentSpec } from "@/lib/types";

describe("buildScheduleAgentSpec", () => {
  it("carries TTLs, task timeout, budgets and tool policy into the schedule spec", () => {
    const values = makeDefaultAgentFormValues({
      sleepTTL: "30m",
      deleteTTL: "24h",
      taskTimeout: "10m",
      budgetUSD: "10.00",
      taskBudgetUSD: "2.50",
      allowedTools: "Read\nmcp__github__*",
      disallowedTools: "Bash, WebFetch",
    });
    const spec = buildScheduleAgentSpec(values);
    expect(spec.sleepTTL).toBe("30m");
    expect(spec.deleteTTL).toBe("24h");
    expect(spec.taskTimeout).toBe("10m");
    expect(spec.budgetUSD).toBe("10.00");
    expect(spec.taskBudgetUSD).toBe("2.50");
    expect(spec.allowedTools).toEqual(["Read", "mcp__github__*"]);
    expect(spec.disallowedTools).toEqual(["Bash", "WebFetch"]);
  });

  it("omits empty TTLs, budgets and tool lists", () => {
    const spec = buildScheduleAgentSpec(makeDefaultAgentFormValues());
    expect(spec.sleepTTL).toBeUndefined();
    expect(spec.deleteTTL).toBeUndefined();
    expect(spec.taskTimeout).toBeUndefined();
    expect(spec.budgetUSD).toBeUndefined();
    expect(spec.taskBudgetUSD).toBeUndefined();
    expect(spec.allowedTools).toBeUndefined();
    expect(spec.disallowedTools).toBeUndefined();
  });
});

describe("effort", () => {
  it("defaults to the 'auto' sentinel, which maps to unset everywhere", () => {
    const values = makeDefaultAgentFormValues();
    expect(values.effort).toBe("auto");
    expect(buildCreateAgentRequest(values).effort).toBeUndefined();
    expect(buildScheduleAgentSpec(values).effort).toBeUndefined();
    expect(buildAgentSpecForSquad(values).effort).toBeUndefined();
  });

  it("passes a concrete effort value through to every payload builder", () => {
    const values = makeDefaultAgentFormValues({ effort: "xhigh" });
    expect(buildCreateAgentRequest(values).effort).toBe("xhigh");
    expect(buildScheduleAgentSpec(values).effort).toBe("xhigh");
    expect(buildAgentSpecForSquad(values).effort).toBe("xhigh");
  });

  it("round-trips an existing schedule's effort back into the 'auto' sentinel when unset", () => {
    const v = agentFormValuesFromScheduleSpec({}, "default");
    expect(v.effort).toBe("auto");
  });

  it("round-trips an existing schedule's concrete effort value", () => {
    const v = agentFormValuesFromScheduleSpec({ effort: "low" }, "default");
    expect(v.effort).toBe("low");
  });
});

describe("buildCreateAgentRequest", () => {
  it("includes tool policy when set", () => {
    const req = buildCreateAgentRequest(
      makeDefaultAgentFormValues({ name: "a", allowedTools: "Read, Write", disallowedTools: "" })
    );
    expect(req.allowedTools).toEqual(["Read", "Write"]);
    expect(req.disallowedTools).toBeUndefined();
  });

  it("omits storage entirely when neither size nor ephemeral is set", () => {
    const req = buildCreateAgentRequest(makeDefaultAgentFormValues({ name: "a" }));
    expect(req.storage).toBeUndefined();
  });

  it("sends storage.ephemeral=true without a size", () => {
    const req = buildCreateAgentRequest(makeDefaultAgentFormValues({ name: "a", ephemeral: true }));
    expect(req.storage).toEqual({ size: undefined, ephemeral: true });
  });

  it("sends both size and ephemeral when both are set", () => {
    const req = buildCreateAgentRequest(
      makeDefaultAgentFormValues({ name: "a", ephemeral: true, storageSize: "10Gi" })
    );
    expect(req.storage).toEqual({ size: "10Gi", ephemeral: true });
  });
});

describe("buildAgentSpecForSquad ephemeral storage", () => {
  it("omits storage when neither size nor ephemeral is set", () => {
    const spec = buildAgentSpecForSquad(makeDefaultAgentFormValues());
    expect(spec.storage).toBeUndefined();
  });

  it("includes ephemeral:true in the squad member's storage", () => {
    const spec = buildAgentSpecForSquad(makeDefaultAgentFormValues({ ephemeral: true }));
    expect(spec.storage).toEqual({ ephemeral: true });
  });

  it("includes both size and ephemeral", () => {
    const spec = buildAgentSpecForSquad(makeDefaultAgentFormValues({ ephemeral: true, storageSize: "5Gi" }));
    expect(spec.storage).toEqual({ size: "5Gi", ephemeral: true });
  });
});

describe("buildCreateAgentRequest", () => {
  it("passes the Archive lifecycle through", () => {
    const req = buildCreateAgentRequest(makeDefaultAgentFormValues({ lifecycle: "Archive" }));
    expect(req.lifecycle).toBe("Archive");
  });
});

describe("agentFormValuesFromScheduleSpec", () => {
  const spec: ScheduleAgentSpec = {
    model: "claude-opus-4-6",
    lifecycle: "AutoDelete",
    role: "manager",
    templateRef: "gpu",
    secrets: ["api-keys"],
    skills: ["python-expert"],
    memories: ["team-context"],
    connectors: ["github"],
    allowedTools: ["Read", "mcp__github__*"],
    disallowedTools: ["Bash"],
    systemPrompt: "Be terse.",
    priority: 5,
    podSpec: {
      containers: [
        { name: "agent", image: "custom:1", resources: { requests: { cpu: "2", memory: "4Gi" }, limits: { cpu: "2", memory: "4Gi" } } },
      ],
    },
    storage: { size: "20Gi", ephemeral: true },
    // The API serialises Go durations with their zero-valued tail units.
    sleepTTL: "30m0s",
    deleteTTL: "24h0m0s",
    taskTimeout: "1h30m0s",
    budgetUSD: "10.00",
    taskBudgetUSD: "2.50",
  };

  it("maps every editable field into form values", () => {
    const v = agentFormValuesFromScheduleSpec(spec, "team-a");
    expect(v.namespace).toBe("team-a");
    expect(v.model).toBe("claude-opus-4-6");
    expect(v.lifecycle).toBe("AutoDelete");
    expect(v.role).toBe("manager");
    expect(v.templateRef).toBe("gpu");
    expect(v.selectedSecretRefs).toEqual(["api-keys"]);
    expect(v.selectedSkills).toEqual(["python-expert"]);
    expect(v.selectedMemories).toEqual(["team-context"]);
    expect(v.selectedConnectors).toEqual(["github"]);
    expect(v.allowedTools).toBe("Read\nmcp__github__*");
    expect(v.disallowedTools).toBe("Bash");
    expect(v.systemPrompt).toBe("Be terse.");
    expect(v.priority).toBe(5);
    expect(v.cpu).toBe("2");
    expect(v.memoryLimit).toBe("4Gi");
    expect(v.image).toBe("custom:1");
    expect(v.storageSize).toBe("20Gi");
    expect(v.ephemeral).toBe(true);
    expect(v.sleepTTL).toBe("30m");
    expect(v.deleteTTL).toBe("24h");
    expect(v.taskTimeout).toBe("1h30m");
    expect(v.budgetUSD).toBe("10.00");
    expect(v.taskBudgetUSD).toBe("2.50");
  });

  it("round-trips through buildScheduleAgentSpec without losing fields", () => {
    const rebuilt = buildScheduleAgentSpec(agentFormValuesFromScheduleSpec(spec, "default"));
    expect(rebuilt).toEqual({ ...spec, sleepTTL: "30m", deleteTTL: "24h", taskTimeout: "1h30m" });
  });

  it("falls back to the form's sentinel defaults when a spec is empty", () => {
    const v = agentFormValuesFromScheduleSpec({}, "default");
    expect(v.lifecycle).toBe("default");
    expect(v.templateRef).toBe("default");
    expect(v.role).toBeUndefined();
    expect(v.selectedSkills).toEqual([]);
    expect(v.allowedTools).toBe("");
  });
});
