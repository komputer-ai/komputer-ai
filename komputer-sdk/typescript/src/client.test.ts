import { describe, it, expect } from "vitest";
import { KomputerClient } from "./client";

describe("KomputerClient", () => {
  it("instantiates with default URL", () => {
    const client = new KomputerClient();
    expect(client).toBeDefined();
  });

  it("instantiates with custom URL", () => {
    const client = new KomputerClient("http://example.com:8080");
    expect(client).toBeDefined();
  });

  it("strips trailing slash from URL", () => {
    const client = new KomputerClient("http://localhost:8080/");
    expect(client).toBeDefined();
  });
});

describe("Agent methods exist", () => {
  const client = new KomputerClient();

  it("has createAgent", () => expect(typeof client.createAgent).toBe("function"));
  it("has listAgents", () => expect(typeof client.listAgents).toBe("function"));
  it("has getAgent", () => expect(typeof client.getAgent).toBe("function"));
  it("has patchAgent", () => expect(typeof client.patchAgent).toBe("function"));
  it("has deleteAgent", () => expect(typeof client.deleteAgent).toBe("function"));
  it("has cancelAgentTask", () => expect(typeof client.cancelAgentTask).toBe("function"));
  it("has getAgentEvents", () => expect(typeof client.getAgentEvents).toBe("function"));
  it("has watchAgent", () => expect(typeof client.watchAgent).toBe("function"));
});

describe("Memory methods exist", () => {
  const client = new KomputerClient();

  it("has createMemory", () => expect(typeof client.createMemory).toBe("function"));
  it("has listMemories", () => expect(typeof client.listMemories).toBe("function"));
  it("has getMemory", () => expect(typeof client.getMemory).toBe("function"));
  it("has patchMemory", () => expect(typeof client.patchMemory).toBe("function"));
  it("has deleteMemory", () => expect(typeof client.deleteMemory).toBe("function"));
});

describe("Skill methods exist", () => {
  const client = new KomputerClient();

  it("has createSkill", () => expect(typeof client.createSkill).toBe("function"));
  it("has listSkills", () => expect(typeof client.listSkills).toBe("function"));
  it("has getSkill", () => expect(typeof client.getSkill).toBe("function"));
  it("has patchSkill", () => expect(typeof client.patchSkill).toBe("function"));
  it("has deleteSkill", () => expect(typeof client.deleteSkill).toBe("function"));
});

describe("Schedule methods exist", () => {
  const client = new KomputerClient();

  it("has createSchedule", () => expect(typeof client.createSchedule).toBe("function"));
  it("has listSchedules", () => expect(typeof client.listSchedules).toBe("function"));
  it("has getSchedule", () => expect(typeof client.getSchedule).toBe("function"));
  it("has patchSchedule", () => expect(typeof client.patchSchedule).toBe("function"));
  it("has deleteSchedule", () => expect(typeof client.deleteSchedule).toBe("function"));
});

describe("Secret methods exist", () => {
  const client = new KomputerClient();

  it("has createSecret", () => expect(typeof client.createSecret).toBe("function"));
  it("has listSecrets", () => expect(typeof client.listSecrets).toBe("function"));
  it("has updateSecret", () => expect(typeof client.updateSecret).toBe("function"));
  it("has deleteSecret", () => expect(typeof client.deleteSecret).toBe("function"));
});

describe("Connector methods exist", () => {
  const client = new KomputerClient();

  it("has createConnector", () => expect(typeof client.createConnector).toBe("function"));
  it("has listConnectors", () => expect(typeof client.listConnectors).toBe("function"));
  it("has getConnector", () => expect(typeof client.getConnector).toBe("function"));
  it("has deleteConnector", () => expect(typeof client.deleteConnector).toBe("function"));
});

describe("Office methods exist", () => {
  const client = new KomputerClient();

  it("has listOffices", () => expect(typeof client.listOffices).toBe("function"));
  it("has getOffice", () => expect(typeof client.getOffice).toBe("function"));
  it("has deleteOffice", () => expect(typeof client.deleteOffice).toBe("function"));
});
