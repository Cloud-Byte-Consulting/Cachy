import { contextBridge, ipcRenderer } from "electron";

contextBridge.exposeInMainWorld("cachyDesktop", {
  getInitialStatus: () => ipcRenderer.invoke("cachy:get-initial-status"),
  getProxyStatus: () => ipcRenderer.invoke("cachy:proxy-status"),
  startProxy: () => ipcRenderer.invoke("cachy:proxy-start"),
  stopProxy: () => ipcRenderer.invoke("cachy:proxy-stop"),
  getProviderConfig: () => ipcRenderer.invoke("cachy:provider-config"),
  saveProviderTarget: (targetBaseURL: string) => ipcRenderer.invoke("cachy:provider-save", targetBaseURL),
  runIntegrationDryRun: (integration: string) => ipcRenderer.invoke("cachy:integration-dry-run", integration),
  getDiagnostics: () => ipcRenderer.invoke("cachy:diagnostics")
});
