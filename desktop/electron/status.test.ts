import { describe, expect, it } from "vitest";
import { getInitialDesktopStatus } from "./status";

describe("getInitialDesktopStatus", () => {
  it("returns a safe startup payload for the renderer", () => {
    const status = getInitialDesktopStatus();

    expect(status.proxyStatus).toBe("not-running");
    expect(JSON.stringify(status)).not.toMatch(/api[_-]?key|admin[_-]?token|secret/i);
  });
});
