import { app, BrowserWindow, ipcMain } from "electron";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { fetchAdminConfig, fetchAdminDiagnostics, fetchAdminStatus, updateAdminConfig } from "./adminClient.js";
import { getInitialDesktopStatus } from "./status.js";
import { CachySupervisor } from "./supervisor.js";
import { providerConfigForRenderer } from "./security.js";

const currentDir = path.dirname(fileURLToPath(import.meta.url));
const supervisor = new CachySupervisor({
  listenAddress: process.env.CACHY_DESKTOP_PROXY_LISTEN,
  targetBaseURL: process.env.CACHY_TARGET_BASE_URL,
  adminStatus: process.env.CACHY_DESKTOP_ADMIN_URL && process.env.CACHY_DESKTOP_ADMIN_TOKEN
    ? () => fetchAdminStatus({
        baseURL: process.env.CACHY_DESKTOP_ADMIN_URL ?? "",
        token: process.env.CACHY_DESKTOP_ADMIN_TOKEN ?? ""
      })
    : undefined
});

function adminOptions() {
  if (!process.env.CACHY_DESKTOP_ADMIN_URL || !process.env.CACHY_DESKTOP_ADMIN_TOKEN) {
    throw new Error("admin API is not configured");
  }
  return {
    baseURL: process.env.CACHY_DESKTOP_ADMIN_URL,
    token: process.env.CACHY_DESKTOP_ADMIN_TOKEN
  };
}

function rendererEntry() {
  return process.env.VITE_DEV_SERVER_URL ?? path.join(currentDir, "../renderer/index.html");
}

export function createMainWindow() {
  const window = new BrowserWindow({
    width: 1180,
    height: 760,
    minWidth: 900,
    minHeight: 620,
    title: "Cachy",
    webPreferences: {
      preload: path.join(currentDir, "preload.cjs"),
      contextIsolation: true,
      nodeIntegration: false
    }
  });

  const entry = rendererEntry();
  if (entry.startsWith("http")) {
    void window.loadURL(entry);
  } else {
    void window.loadFile(entry);
  }
  return window;
}

ipcMain.handle("cachy:get-initial-status", () => getInitialDesktopStatus());
ipcMain.handle("cachy:proxy-status", () => supervisor.status());
ipcMain.handle("cachy:proxy-start", () => supervisor.start());
ipcMain.handle("cachy:proxy-stop", () => supervisor.stop());
ipcMain.handle("cachy:provider-config", async () => {
  const config = await fetchAdminConfig(adminOptions());
  return providerConfigForRenderer(config);
});
ipcMain.handle("cachy:provider-save", async (_event, targetBaseURL: string) => {
  const config = await updateAdminConfig(adminOptions(), targetBaseURL);
  return providerConfigForRenderer(config);
});
ipcMain.handle("cachy:integration-dry-run", (_event, integration: string) => {
  switch (integration) {
    case "codex":
      return "cachy integrations codex repair --dry-run";
    case "claude":
      return "cachy integrations claude repair --dry-run";
    case "mcp":
      return "cachy integrations mcp --target <provider-url> --listen 127.0.0.1:8787";
    default:
      throw new Error("unsupported integration");
  }
});
ipcMain.handle("cachy:diagnostics", async () => {
  const diagnostics = await fetchAdminDiagnostics(adminOptions());
  return {
    health: diagnostics.health,
    proxyListenAddress: diagnostics.proxy_listen_address,
    configDir: diagnostics.config_dir,
    recentFailureCategories: diagnostics.recent_failure_categories
  };
});

void app.whenReady().then(() => {
  createMainWindow();

  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createMainWindow();
    }
  });
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") {
    app.quit();
  }
});
