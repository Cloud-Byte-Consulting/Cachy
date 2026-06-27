export type ProxyStatus = "not-running" | "starting" | "running" | "error" | "disabled";

export type DesktopStatus = {
  proxyStatus: ProxyStatus;
  activeProvider: string;
  savingsToday: string;
  recentFailures: string[];
  nextAction: string;
};

export type RuntimeStatusView = {
  state: "stopped" | "running" | "error";
  binary?: {
    path: string;
    source: "installed" | "bundled";
  };
  lastError?: string;
  admin?: {
    status?: string;
    version?: string;
    target_base_url?: string;
    proxy_listen_address?: string;
  };
};

export type DashboardState =
  | { kind: "loading"; status: DesktopStatus; message: string }
  | { kind: "empty"; status: DesktopStatus; runtime?: RuntimeStatusView }
  | { kind: "healthy"; status: DesktopStatus; runtime: RuntimeStatusView }
  | { kind: "error"; status: DesktopStatus; message: string; runtime?: RuntimeStatusView }
  | { kind: "disabled"; status: DesktopStatus; message: string }
  | { kind: "permission-denied"; status: DesktopStatus; message: string };

export function createInitialStatus(overrides: Partial<DesktopStatus> = {}): DesktopStatus {
  return {
    proxyStatus: "not-running",
    activeProvider: "No provider connected",
    savingsToday: "0 tokens",
    recentFailures: [],
    nextAction: "Start Cachy or connect to an existing proxy",
    ...overrides
  };
}

export function createLoadingState(): DashboardState {
  return {
    kind: "loading",
    status: createInitialStatus({ nextAction: "Checking Cachy status..." }),
    message: "Checking Cachy status..."
  };
}

export function createDisabledState(): DashboardState {
  return {
    kind: "disabled",
    status: createInitialStatus({
      proxyStatus: "disabled",
      nextAction: "Open the desktop app with Electron preload enabled"
    }),
    message: "Desktop controls unavailable"
  };
}

export function dashboardStateFromRuntime(runtime: RuntimeStatusView): DashboardState {
  const target = runtime.admin?.target_base_url ?? "No provider connected";
  if (runtime.lastError?.match(/401|403|permission|denied/i)) {
    return {
      kind: "permission-denied",
      status: createInitialStatus({
        proxyStatus: "error",
        activeProvider: target,
        recentFailures: [runtime.lastError],
        nextAction: "Reconnect with a valid admin token"
      }),
      message: "Admin access denied"
    };
  }
  if (runtime.state === "error") {
    const message = runtime.lastError ?? "Cachy reported an error";
    return {
      kind: "error",
      status: createInitialStatus({
        proxyStatus: "error",
        activeProvider: target,
        recentFailures: [message],
        nextAction: "Review diagnostics and restart the proxy"
      }),
      message,
      runtime
    };
  }
  if (runtime.state === "running") {
    return {
      kind: "healthy",
      status: createInitialStatus({
        proxyStatus: "running",
        activeProvider: target,
        nextAction: "Monitor traffic and recent failures"
      }),
      runtime
    };
  }
  return {
    kind: "empty",
    status: createInitialStatus({
      activeProvider: target,
      nextAction: "Start the proxy or choose a provider target"
    }),
    runtime
  };
}
