import { readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const desktopDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const preloadJSPath = path.join(desktopDir, "dist", "electron", "preload.js");
const preloadCJSPath = path.join(desktopDir, "dist", "electron", "preload.cjs");

const source = readFileSync(preloadJSPath, "utf8");
const commonJS = source.replace(
  'import { contextBridge, ipcRenderer } from "electron";',
  'const { contextBridge, ipcRenderer } = require("electron");'
);

writeFileSync(preloadCJSPath, commonJS);
