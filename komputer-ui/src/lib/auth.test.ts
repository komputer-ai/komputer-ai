// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from "vitest";
import { authHeaders, getApiToken, setApiToken, withWsToken } from "./auth";

describe("api token", () => {
  beforeEach(() => setApiToken(""));

  it("sends no header without a token", () => {
    expect(authHeaders()).toEqual({});
    expect(withWsToken("ws://x/api/v1/agents/a/ws")).toBe("ws://x/api/v1/agents/a/ws");
  });

  it("sends the saved token", () => {
    setApiToken("s3cret");
    expect(getApiToken()).toBe("s3cret");
    expect(authHeaders()).toEqual({ Authorization: "Bearer s3cret" });
  });

  it("adds the token to WebSocket URLs", () => {
    setApiToken("a b&c");
    expect(withWsToken("ws://x/ws")).toBe("ws://x/ws?access_token=a%20b%26c");
    expect(withWsToken("ws://x/ws?n=1")).toBe("ws://x/ws?n=1&access_token=a%20b%26c");
  });
});
