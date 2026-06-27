import { spawn as spawnProcess } from "node:child_process";
import path from "node:path";
import { existsSync } from "node:fs";
import type { AdminStatus } from "./adminClient.js";

export type BinarySource = "installed" | "bundled";

export type BinaryCandidate = {
  path: string;
  source: BinarySource;
};

export type SupervisorState = "stopped" | "running" | "error";

export type SupervisedProcess = {
  pid?: number;
  kill: (signal?: NodeJS.Signals | number) => boolean;
  once: (event: "exit" | "error", listener: (...args: unknown[]) => void) => SupervisedProcess;
};

export type SpawnProcess = (
  command: string,
  args: string[],
  options: { stdio: "ignore"; windowsHide: boolean }
) => SupervisedProcess;

export type DiscoveryOptions = {
  env?: Record<string, string | undefined>;
  platform?: NodeJS.Platform;
  arch?: string;
  resourcesPath?: string;
  pathExists?: (candidate: string) => boolean;
  pathDelimiter?: string;
};

export type SupervisorOptions = {
  discoverBinary?: () => BinaryCandidate | null;
  spawn?: SpawnProcess;
  readVersion?: (binaryPath: string) => Promise<string>;
  expectedVersion?: string;
  listenAddress?: string;
  targetBaseURL?: string;
  adminStatus?: () => Promise<AdminStatus>;
};

export type RuntimeStatus = {
  state: SupervisorState;
  binary?: BinaryCandidate;
  pid?: number;
  lastError?: string;
  admin?: AdminStatus;
};

const defaultListenAddress = "127.0.0.1:8787";

export function discoverCachyBinary(options: DiscoveryOptions = {}): BinaryCandidate | null {
  const platform = options.platform ?? process.platform;
  const arch = options.arch ?? process.arch;
  const env = options.env ?? process.env;
  const pathExists = options.pathExists ?? existsSync;
  const pathDelimiter = options.pathDelimiter ?? path.delimiter;
  const binaryName = platform === "win32" ? "cachy.exe" : "cachy";

  for (const entry of (env.PATH ?? "").split(pathDelimiter)) {
    if (!entry) {
      continue;
    }
    const candidate = joinForPlatform(platform, entry, binaryName);
    if (pathExists(candidate)) {
      return { path: candidate, source: "installed" };
    }
  }

  const resourcesPath = options.resourcesPath ?? process.resourcesPath;
  if (resourcesPath) {
    const bundled = joinForPlatform(platform, resourcesPath, "bin", `${platform}-${arch}`, binaryName);
    if (pathExists(bundled)) {
      return { path: bundled, source: "bundled" };
    }
  }
  return null;
}

function joinForPlatform(platform: NodeJS.Platform, ...parts: string[]) {
  return platform === "win32" ? path.win32.join(...parts) : path.posix.join(...parts);
}

export class CachySupervisor {
  private readonly discoverBinary: () => BinaryCandidate | null;
  private readonly spawn: SpawnProcess;
  private readonly readVersion?: (binaryPath: string) => Promise<string>;
  private readonly expectedVersion?: string;
  private readonly listenAddress: string;
  private readonly targetBaseURL?: string;
  private readonly adminStatus?: () => Promise<AdminStatus>;
  private process?: SupervisedProcess;
  private binary?: BinaryCandidate;
  private state: SupervisorState = "stopped";
  private lastError?: string;

  constructor(options: SupervisorOptions = {}) {
    this.discoverBinary = options.discoverBinary ?? (() => discoverCachyBinary());
    this.spawn = options.spawn ?? ((command, args, spawnOptions) => spawnProcess(command, args, spawnOptions));
    this.readVersion = options.readVersion;
    this.expectedVersion = options.expectedVersion;
    this.listenAddress = options.listenAddress ?? defaultListenAddress;
    this.targetBaseURL = options.targetBaseURL;
    this.adminStatus = options.adminStatus;
  }

  async start(): Promise<RuntimeStatus> {
    if (this.process && this.state === "running") {
      return this.status();
    }

    const binary = this.discoverBinary();
    if (!binary) {
      return this.fail("cachy binary was not found");
    }

    if (this.expectedVersion && this.readVersion) {
      const version = await this.readVersion(binary.path);
      if (version !== this.expectedVersion) {
        return this.fail(`cachy binary version ${version} does not match ${this.expectedVersion}`);
      }
    }

    const args = ["proxy", "--listen", this.listenAddress];
    if (this.targetBaseURL) {
      args.push("--target", this.targetBaseURL);
    }

    try {
      const child = this.spawn(binary.path, args, { stdio: "ignore", windowsHide: true });
      this.process = child;
      this.binary = binary;
      this.state = "running";
      this.lastError = undefined;
      child.once("exit", () => {
        if (this.process === child) {
          this.process = undefined;
          this.state = "stopped";
        }
      });
      child.once("error", (error) => {
        if (this.process === child) {
          this.fail(error instanceof Error ? error.message : "cachy process failed");
        }
      });
      return this.status();
    } catch (error) {
      return this.fail(error instanceof Error ? error.message : "failed to start cachy");
    }
  }

  async stop(): Promise<RuntimeStatus> {
    if (!this.process) {
      this.state = "stopped";
      return this.status();
    }
    const stopped = this.process.kill();
    if (!stopped) {
      return this.fail("failed to stop cachy process");
    }
    this.process = undefined;
    this.state = "stopped";
    return this.status();
  }

  async restart(): Promise<RuntimeStatus> {
    await this.stop();
    return this.start();
  }

  async status(): Promise<RuntimeStatus> {
    const status: RuntimeStatus = {
      state: this.state,
      binary: this.binary,
      pid: this.process?.pid,
      lastError: this.lastError
    };
    if (this.adminStatus) {
      try {
        status.admin = await this.adminStatus();
      } catch (error) {
        status.lastError = error instanceof Error ? error.message : "admin status unavailable";
      }
    }
    return status;
  }

  private fail(message: string): RuntimeStatus {
    this.state = "error";
    this.lastError = message;
    return { state: this.state, binary: this.binary, pid: this.process?.pid, lastError: this.lastError };
  }
}
