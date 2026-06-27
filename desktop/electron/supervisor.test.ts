import { describe, expect, it } from "vitest";
import { CachySupervisor, discoverCachyBinary, type SupervisedProcess } from "./supervisor";

class FakeProcess implements SupervisedProcess {
  pid?: number;
  stopped = false;
  private listeners = new Map<string, (...args: unknown[]) => void>();

  constructor(pid: number, private readonly killResult = true) {
    this.pid = pid;
  }

  kill() {
    this.stopped = true;
    return this.killResult;
  }

  once(event: "exit" | "error", listener: (...args: unknown[]) => void) {
    this.listeners.set(event, listener);
    return this;
  }

  emit(event: "exit" | "error", value?: unknown) {
    this.listeners.get(event)?.(value);
  }
}

describe("discoverCachyBinary", () => {
  it("prefers an installed cachy binary before bundled resources", () => {
    const found = discoverCachyBinary({
      env: { PATH: ["/usr/local/bin", "/opt/bin"].join(":") },
      platform: "linux",
      arch: "x64",
      resourcesPath: "/app/resources",
      pathDelimiter: ":",
      pathExists: (candidate) => candidate === "/opt/bin/cachy" || candidate === "/app/resources/bin/linux-x64/cachy"
    });

    expect(found).toEqual({ path: "/opt/bin/cachy", source: "installed" });
  });

  it("falls back to the bundled binary", () => {
    const found = discoverCachyBinary({
      env: { PATH: "/usr/local/bin" },
      platform: "win32",
      arch: "x64",
      resourcesPath: "C:\\Cachy\\resources",
      pathDelimiter: ";",
      pathExists: (candidate) => candidate.endsWith("bin\\win32-x64\\cachy.exe")
    });

    expect(found?.source).toBe("bundled");
  });

  it("returns null when no binary is available", () => {
    expect(discoverCachyBinary({ env: { PATH: "" }, pathExists: () => false })).toBeNull();
  });
});

describe("CachySupervisor", () => {
  it("reports missing binaries without spawning", async () => {
    const supervisor = new CachySupervisor({
      discoverBinary: () => null,
      spawn: () => {
        throw new Error("spawn should not run");
      }
    });

    await expect(supervisor.start()).resolves.toMatchObject({
      state: "error",
      lastError: "cachy binary was not found"
    });
  });

  it("rejects an unexpected binary version before spawning", async () => {
    const supervisor = new CachySupervisor({
      discoverBinary: () => ({ path: "/usr/local/bin/cachy", source: "installed" }),
      expectedVersion: "1.0.0",
      readVersion: async () => "0.9.0",
      spawn: () => {
        throw new Error("spawn should not run");
      }
    });

    await expect(supervisor.start()).resolves.toMatchObject({
      state: "error",
      lastError: "cachy binary version 0.9.0 does not match 1.0.0"
    });
  });

  it("starts cachy with proxy arguments and includes admin status", async () => {
    const spawned: Array<{ command: string; args: string[] }> = [];
    const process = new FakeProcess(42);
    const supervisor = new CachySupervisor({
      discoverBinary: () => ({ path: "/usr/local/bin/cachy", source: "installed" }),
      targetBaseURL: "http://127.0.0.1:11434",
      adminStatus: async () => ({ status: "running", version: "dev" }),
      spawn: (command, args) => {
        spawned.push({ command, args });
        return process;
      }
    });

    await expect(supervisor.start()).resolves.toMatchObject({
      state: "running",
      pid: 42,
      admin: { status: "running", version: "dev" }
    });
    expect(spawned).toEqual([
      {
        command: "/usr/local/bin/cachy",
        args: ["proxy", "--listen", "127.0.0.1:8787", "--target", "http://127.0.0.1:11434"]
      }
    ]);
  });

  it("reports start failures", async () => {
    const supervisor = new CachySupervisor({
      discoverBinary: () => ({ path: "/usr/local/bin/cachy", source: "installed" }),
      spawn: () => {
        throw new Error("permission denied");
      }
    });

    await expect(supervisor.start()).resolves.toMatchObject({
      state: "error",
      lastError: "permission denied"
    });
  });

  it("reports stop failures", async () => {
    const supervisor = new CachySupervisor({
      discoverBinary: () => ({ path: "/usr/local/bin/cachy", source: "installed" }),
      spawn: () => new FakeProcess(42, false)
    });

    await supervisor.start();
    await expect(supervisor.stop()).resolves.toMatchObject({
      state: "error",
      lastError: "failed to stop cachy process"
    });
  });

  it("restarts by stopping the existing process and spawning again", async () => {
    const processes = [new FakeProcess(1), new FakeProcess(2)];
    const supervisor = new CachySupervisor({
      discoverBinary: () => ({ path: "/usr/local/bin/cachy", source: "installed" }),
      spawn: () => processes.shift() ?? new FakeProcess(99)
    });

    await supervisor.start();
    await expect(supervisor.restart()).resolves.toMatchObject({ state: "running", pid: 2 });
    expect(processes).toHaveLength(0);
  });
});
