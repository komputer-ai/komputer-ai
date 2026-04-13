/**
 * Integration tests for the KomputerClient.
 *
 * These tests require a running komputer.ai API.
 * Set KOMPUTER_API_URL to point to the server (default: http://localhost:8080).
 *
 * Run with:
 *   npx vitest run src/integration.test.ts
 */

import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { KomputerClient } from "./client";

declare const process: { env: Record<string, string | undefined> };
const BASE_URL: string = process.env.KOMPUTER_API_URL || "http://localhost:8080";
const client = new KomputerClient(BASE_URL);

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async function tryDelete(fn: () => Promise<unknown>): Promise<void> {
  try {
    await fn();
  } catch {
    // ignore — resource may not exist
  }
}

// ---------------------------------------------------------------------------
// Memories CRUD
// ---------------------------------------------------------------------------

describe("Memories CRUD", () => {
  const name = "sdk-test-ts-memory-crud";

  beforeAll(async () => {
    await tryDelete(() => client.deleteMemory(name));
  });

  afterAll(async () => {
    await tryDelete(() => client.deleteMemory(name));
  });

  it("creates a memory", async () => {
    const mem = await client.createMemory({
      name,
      content: "initial content",
      description: "integration test memory",
    });
    expect(mem).toBeDefined();
    expect((mem as any).name).toBe(name);
  });

  it("gets the created memory", async () => {
    const mem = await client.getMemory(name);
    expect(mem).toBeDefined();
    expect((mem as any).name).toBe(name);
  });

  it("lists memories and finds the created one", async () => {
    const list = await client.listMemories();
    const items: any[] = Array.isArray(list) ? list : (list as any).items ?? [];
    const found = items.some((m: any) => m.name === name);
    expect(found).toBe(true);
  });

  it("patches the memory", async () => {
    const patched = await client.patchMemory({
      name,
      content: "patched content",
    });
    expect(patched).toBeDefined();
  });

  it("get reflects the patched content", async () => {
    const mem = await client.getMemory(name);
    const content: string = (mem as any).content ?? (mem as any).spec?.content ?? "";
    expect(content).toContain("patched");
  });

  it("deletes the memory", async () => {
    await expect(client.deleteMemory(name)).resolves.not.toThrow();
  });

  it("get throws after delete", async () => {
    await expect(client.getMemory(name)).rejects.toThrow();
  });
});

// ---------------------------------------------------------------------------
// Memories idempotent create
// ---------------------------------------------------------------------------

describe("Memories idempotent create", () => {
  const name = "sdk-test-ts-memory-idempotent";

  beforeAll(async () => {
    await tryDelete(() => client.deleteMemory(name));
  });

  afterAll(async () => {
    await tryDelete(() => client.deleteMemory(name));
  });

  it("creates a memory the first time", async () => {
    await expect(
      client.createMemory({ name, content: "first content" })
    ).resolves.not.toThrow();
  });

  it("creates the same memory again with different content without error", async () => {
    await expect(
      client.createMemory({ name, content: "second content" })
    ).resolves.not.toThrow();
  });

  it("get reflects the updated content", async () => {
    const mem = await client.getMemory(name);
    const content: string = (mem as any).content ?? (mem as any).spec?.content ?? "";
    expect(content).toContain("second");
  });
});

// ---------------------------------------------------------------------------
// Skills CRUD
// ---------------------------------------------------------------------------

describe("Skills CRUD", () => {
  const name = "sdk-test-ts-skill-crud";

  beforeAll(async () => {
    await tryDelete(() => client.deleteSkill(name));
  });

  afterAll(async () => {
    await tryDelete(() => client.deleteSkill(name));
  });

  it("creates a skill", async () => {
    const skill = await client.createSkill({
      name,
      content: "echo hello",
      description: "integration test skill",
    });
    expect(skill).toBeDefined();
    expect((skill as any).name).toBe(name);
  });

  it("gets the created skill", async () => {
    const skill = await client.getSkill(name);
    expect(skill).toBeDefined();
    expect((skill as any).name).toBe(name);
  });

  it("lists skills and finds the created one", async () => {
    const list = await client.listSkills();
    const items: any[] = Array.isArray(list) ? list : (list as any).items ?? [];
    const found = items.some((s: any) => s.name === name);
    expect(found).toBe(true);
  });

  it("patches the skill", async () => {
    const patched = await client.patchSkill({
      name,
      content: "echo patched",
      description: "patched description",
    });
    expect(patched).toBeDefined();
  });

  it("get reflects the patched content", async () => {
    const skill = await client.getSkill(name);
    const content: string = (skill as any).content ?? (skill as any).spec?.content ?? "";
    expect(content).toContain("patched");
  });

  it("deletes the skill", async () => {
    await expect(client.deleteSkill(name)).resolves.not.toThrow();
  });

  it("get throws after delete", async () => {
    await expect(client.getSkill(name)).rejects.toThrow();
  });
});

// ---------------------------------------------------------------------------
// Skills idempotent create
// ---------------------------------------------------------------------------

describe("Skills idempotent create", () => {
  const name = "sdk-test-ts-skill-idempotent";

  beforeAll(async () => {
    await tryDelete(() => client.deleteSkill(name));
  });

  afterAll(async () => {
    await tryDelete(() => client.deleteSkill(name));
  });

  it("creates a skill the first time", async () => {
    await expect(
      client.createSkill({ name, content: "echo first", description: "first" })
    ).resolves.not.toThrow();
  });

  it("creates the same skill again with different content without error", async () => {
    await expect(
      client.createSkill({ name, content: "echo second", description: "second" })
    ).resolves.not.toThrow();
  });

  it("get reflects the updated content", async () => {
    const skill = await client.getSkill(name);
    const content: string = (skill as any).content ?? (skill as any).spec?.content ?? "";
    expect(content).toContain("second");
  });
});

// ---------------------------------------------------------------------------
// Secrets CRUD
// ---------------------------------------------------------------------------

describe("Secrets CRUD", () => {
  const name = "sdk-test-ts-secret-crud";

  beforeAll(async () => {
    await tryDelete(() => client.deleteSecret(name));
  });

  afterAll(async () => {
    await tryDelete(() => client.deleteSecret(name));
  });

  it("creates a secret", async () => {
    const secret = await client.createSecret({
      name,
      data: { MY_KEY: "my-value" },
    });
    expect(secret).toBeDefined();
    expect((secret as any).name).toBe(name);
  });

  it("lists secrets and finds the created one", async () => {
    const list = await client.listSecrets();
    const items: any[] = Array.isArray(list) ? list : (list as any).items ?? [];
    const found = items.some((s: any) => s.name === name);
    expect(found).toBe(true);
  });

  it("updates the secret", async () => {
    const updated = await client.updateSecret({
      name,
      data: { MY_KEY: "updated-value" },
    });
    expect(updated).toBeDefined();
  });

  it("deletes the secret", async () => {
    await expect(client.deleteSecret(name)).resolves.not.toThrow();
  });
});

// ---------------------------------------------------------------------------
// Secrets idempotent create
// ---------------------------------------------------------------------------

describe("Secrets idempotent create", () => {
  const name = "sdk-test-ts-secret-idempotent";

  beforeAll(async () => {
    await tryDelete(() => client.deleteSecret(name));
  });

  afterAll(async () => {
    await tryDelete(() => client.deleteSecret(name));
  });

  it("creates a secret the first time", async () => {
    await expect(
      client.createSecret({ name, data: { TOKEN: "first-token" } })
    ).resolves.not.toThrow();
  });

  it("creates the same secret again with different data without error", async () => {
    await expect(
      client.createSecret({ name, data: { TOKEN: "second-token" } })
    ).resolves.not.toThrow();
  });

  it("lists secrets and still finds the entry", async () => {
    const list = await client.listSecrets();
    const items: any[] = Array.isArray(list) ? list : (list as any).items ?? [];
    const found = items.some((s: any) => s.name === name);
    expect(found).toBe(true);
  });
});
