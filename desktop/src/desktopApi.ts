import type { RuntimeStatusView } from "./status";

export type DesktopAPI = {
  getProxyStatus: () => Promise<RuntimeStatusView>;
  startProxy: () => Promise<RuntimeStatusView>;
  stopProxy: () => Promise<RuntimeStatusView>;
  getProviderConfig?: () => Promise<ProviderConfigView>;
  saveProviderTarget?: (targetBaseURL: string) => Promise<ProviderConfigView>;
  runIntegrationDryRun?: (integration: IntegrationKind) => Promise<string>;
  getDiagnostics?: () => Promise<DiagnosticsView>;
};

export type ProviderConfigView = {
  targetBaseURL?: string;
  credentialState?: string;
};

export type IntegrationKind = "codex" | "claude" | "mcp";

export type DiagnosticsView = {
  health?: string;
  proxyListenAddress?: string;
  configDir?: string;
  recentFailureCategories?: Record<string, number>;
};

declare global {
  interface Window {
    cachyDesktop?: DesktopAPI & {
      getInitialStatus?: () => Promise<unknown>;
    };
  }
}

export function getDesktopAPI(): DesktopAPI | undefined {
  return window.cachyDesktop;
}
