import { describe, it, expect } from "vitest";
import {
  makeDefaultAgentFormValues,
  buildScheduleAgentSpec,
  agentFormValuesFromScheduleSpec,
  buildCreateAgentRequest,
} from "./agent-fields-form";
import type { ScheduleAgentSpec } from "@/lib/types";

describe("buildScheduleAgentSpec", () => {
  it("carries TTLs, task timeout and tool policy into the schedule spec", () => {
    const values = makeDefaultAgentFormValues({
      sleepTTL: "30m",
      deleteTTL: "24h",
      taskTimeout: "10m",
      allowedTools: "Read\nmcp__github__*",
      disallowedTools: "Bash, WebFetch",
    });
    const spec = buildScheduleAgentSpec(values);
    expect(spec.sleepTTL).toBe("30m");
    expect(spec.deleteTTL).toBe("24h");
    expect(spec.taskTimeout).toBe("10m");
    expect(spec.allowedTools).toEqual(["Read", "mcp__github__*"]);
    expect(spec.disallowedTools).toEqual(["Bash", "WebFetch"]);
  });

  it("omits empty TTLs and tool lists", () => {
    const spec = buildScheduleAgentSpec(makeDefaultAgentFormValues());
    expect(spec.sleepTTL).toBeUndefined();
    expect(spec.deleteTTL).toBeUndefined();
    expect(spec.taskTimeout).toBeUndefined();
    expect(spec.allowedTools).toBeUndefined();
    expect(spec.disallowedTools).toBeUndefined();
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
    storage: { size: "20Gi" },
    // The API serialises Go durations with their zero-valued tail units.
    sleepTTL: "30m0s",
    deleteTTL: "24h0m0s",
    taskTimeout: "1h30m0s",
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
    expect(v.sleepTTL).toBe("30m");
    expect(v.deleteTTL).toBe("24h");
    expect(v.taskTimeout).toBe("1h30m");
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
