import { _electron as electron, expect, test, type ElectronApplication } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { createServer, type IncomingMessage, type Server, type ServerResponse } from "node:http";
import type { AddressInfo } from "node:net";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const desktopDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = path.resolve(desktopDir, "..");
const adminToken = "e2e-admin-token";
const providerSecret = "sk-e2e-provider-secret";
const providerTarget = "http://127.0.0.1:11434";

type AdminFixture = {
  url: string;
  updates: string[];
  close: () => Promise<void>;
};

test("starts the live desktop app against admin fixtures without exposing secrets", async () => {
  const admin = await startAdminFixture();
  const binary = buildCachyBinary();
  const proxyListen = await freeListenAddress();
  let app: ElectronApplication | undefined;

  try {
    app = await launchApp(admin.url, binary.dir, proxyListen, adminToken);
    const page = await app.firstWindow();

    await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
    await expect(page.getByText("Credential: present")).toBeVisible();

    await page.getByRole("button", { name: "Start Proxy" }).click();

    await expect(page.getByRole("button", { name: "Stop Proxy" })).toBeVisible();
    await expect(page.locator(".status")).toContainText("running");
    await expect(page.getByText(providerTarget).first()).toBeVisible();

    const body = await page.locator("body").innerText();
    expect(body).not.toContain(adminToken);
    expect(body).not.toContain(providerSecret);
  } finally {
    await app?.close();
    await admin.close();
    rmSync(binary.dir, { recursive: true, force: true });
  }
});

test("saves provider target through the live UI", async () => {
  const admin = await startAdminFixture();
  const binary = buildCachyBinary();
  const proxyListen = await freeListenAddress();
  let app: ElectronApplication | undefined;

  try {
    app = await launchApp(admin.url, binary.dir, proxyListen, adminToken);
    const page = await app.firstWindow();
    const nextTarget = "http://127.0.0.1:12345";

    await page.getByLabel("Target URL").fill(nextTarget);
    await page.getByRole("button", { name: "Save Provider" }).click();

    await expect(page.getByText("Provider target saved")).toBeVisible();
    expect(admin.updates).toEqual([nextTarget]);
  } finally {
    await app?.close();
    await admin.close();
    rmSync(binary.dir, { recursive: true, force: true });
  }
});

test("shows admin permission failures without exposing the configured token", async () => {
  const admin = await startAdminFixture({ expectedToken: "different-token" });
  const binary = buildCachyBinary();
  const proxyListen = await freeListenAddress();
  let app: ElectronApplication | undefined;

  try {
    app = await launchApp(admin.url, binary.dir, proxyListen, adminToken);
    const page = await app.firstWindow();

    await expect(page.getByText("Admin access denied")).toBeVisible();
    await page.getByRole("button", { name: "Run Diagnostics" }).click();
    await expect(page.getByText("admin diagnostics request failed with 401")).toBeVisible();

    const body = await page.locator("body").innerText();
    expect(body).not.toContain(adminToken);
  } finally {
    await app?.close();
    await admin.close();
    rmSync(binary.dir, { recursive: true, force: true });
  }
});

async function launchApp(adminURL: string, binaryDir: string, proxyListen: string, token: string) {
  return electron.launch({
    args: [path.join(desktopDir, "dist", "electron", "main.js")],
    cwd: desktopDir,
    env: {
      ...process.env,
      CACHY_DESKTOP_ADMIN_TOKEN: token,
      CACHY_DESKTOP_ADMIN_URL: adminURL,
      CACHY_DESKTOP_PROXY_LISTEN: proxyListen,
      CACHY_TARGET_BASE_URL: providerTarget,
      PATH: `${binaryDir}${path.delimiter}${process.env.PATH ?? ""}`
    }
  });
}

function buildCachyBinary() {
  const dir = mkdtempSync(path.join(tmpdir(), "cachy-e2e-bin-"));
  const binaryName = process.platform === "win32" ? "cachy.exe" : "cachy";
  const binaryPath = path.join(dir, binaryName);
  execFileSync("go", ["build", "-trimpath", "-o", binaryPath, "./cmd/cachy"], {
    cwd: repoRoot,
    env: { ...process.env, CGO_ENABLED: "0" },
    stdio: "inherit"
  });
  return { dir, binaryPath };
}

async function startAdminFixture(options: { expectedToken?: string } = {}): Promise<AdminFixture> {
  const expectedToken = options.expectedToken ?? adminToken;
  const updates: string[] = [];
  const server = createServer(async (req, res) => {
    if (req.headers.authorization !== `Bearer ${expectedToken}`) {
      writeJSON(res, 401, { error: "unauthorized" });
      return;
    }

    if (req.method === "GET" && req.url === "/admin/v1/status") {
      writeJSON(res, 200, {
        status: "running",
        version: "e2e",
        proxy_listen_address: "127.0.0.1:8787",
        target_base_url: providerTarget
      });
      return;
    }

    if (req.method === "GET" && req.url === "/admin/v1/config") {
      writeJSON(res, 200, {
        target_base_url: providerTarget,
        provider_credential: providerSecret
      });
      return;
    }

    if (req.method === "PUT" && req.url === "/admin/v1/config") {
      const body = await readBody(req);
      const payload = JSON.parse(body) as { target_base_url?: string };
      updates.push(payload.target_base_url ?? "");
      writeJSON(res, 200, {
        target_base_url: payload.target_base_url,
        provider_credential: "<redacted>"
      });
      return;
    }

    if (req.method === "GET" && req.url === "/admin/v1/diagnostics") {
      writeJSON(res, 200, {
        health: "ok",
        proxy_listen_address: "127.0.0.1:8787",
        recent_failure_categories: { upstream: 2 }
      });
      return;
    }

    writeJSON(res, 404, { error: "not found" });
  });

  await listen(server);
  const address = server.address() as AddressInfo;
  return {
    url: `http://127.0.0.1:${address.port}`,
    updates,
    close: () => close(server)
  };
}

async function freeListenAddress() {
  const server = createServer();
  await listen(server);
  const address = server.address() as AddressInfo;
  await close(server);
  return `127.0.0.1:${address.port}`;
}

function listen(server: Server) {
  return new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      server.off("error", reject);
      resolve();
    });
  });
}

function close(server: Server) {
  return new Promise<void>((resolve, reject) => {
    server.close((error) => {
      if (error) {
        reject(error);
        return;
      }
      resolve();
    });
  });
}

function readBody(req: IncomingMessage) {
  return new Promise<string>((resolve, reject) => {
    let body = "";
    req.setEncoding("utf8");
    req.on("data", (chunk) => {
      body += chunk;
    });
    req.on("end", () => resolve(body));
    req.on("error", reject);
  });
}

function writeJSON(res: ServerResponse, status: number, value: unknown) {
  res.writeHead(status, { "content-type": "application/json" });
  res.end(JSON.stringify(value));
}
