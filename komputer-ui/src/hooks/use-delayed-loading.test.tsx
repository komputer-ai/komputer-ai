// @vitest-environment jsdom
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useDelayedLoading } from "./use-delayed-loading";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

function Probe({ loading }: { loading: boolean }) {
  return <span>{useDelayedLoading(loading) ? "skeleton" : "blank"}</span>;
}

describe("useDelayedLoading", () => {
  let el: HTMLDivElement;
  beforeEach(() => {
    vi.useFakeTimers();
    el = document.createElement("div");
  });
  afterEach(() => vi.useRealTimers());

  it("shows the loading state soon after a load starts, not seconds later", () => {
    const root = createRoot(el);
    act(() => root.render(<Probe loading />));
    act(() => vi.advanceTimersByTime(250));
    expect(el.textContent).toBe("blank"); // a fast load doesn't flash a skeleton
    act(() => vi.advanceTimersByTime(100));
    expect(el.textContent).toBe("skeleton");
    act(() => root.render(<Probe loading={false} />));
    expect(el.textContent).toBe("blank");
    act(() => root.unmount());
  });
});
