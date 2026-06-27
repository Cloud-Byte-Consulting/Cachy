import { describe, expect, it } from "vitest";
import { fetchAdminConfig, fetchAdminDiagnostics, fetchAdminStatus, updateAdminConfig } from "./adminClient";

describe("fetchAdminStatus", () => {
  it("fetches status from the admin API with bearer auth", async () => {
    const calls: Array<{ input: string; authorization?: string }> = [];

    const status = await fetchAdminStatus({
      baseURL: "http://127.0.0.1:9191/",
      token: "admin-token",
      fetchImpl: async (input, init) => {
        calls.push({ input, authorization: init?.headers?.Authorization });
        return {
          ok: true,
          status: 200,
          json: async () => ({ status: "running", version: "dev" })
        };
      }
    });

    expect(status.status).toBe("running");
    expect(calls).toEqual([
      {
        input: "http://127.0.0.1:9191/admin/v1/status",
        authorization: "Bearer admin-token"
      }
    ]);
  });

  it("reports admin API failures without echoing tokens", async () => {
    await expect(
      fetchAdminStatus({
        baseURL: "http://127.0.0.1:9191",
        token: "secret-token",
        fetchImpl: async () => ({
          ok: false,
          status: 401,
          json: async () => ({})
        })
      })
    ).rejects.toThrow("admin status request failed with 401");
  });

  it("reports config auth failures without echoing tokens", async () => {
    await expect(
      fetchAdminConfig({
        baseURL: "http://127.0.0.1:9191",
        token: "secret-token",
        fetchImpl: async () => ({
          ok: false,
          status: 403,
          json: async () => ({})
        })
      })
    ).rejects.toThrow("admin config request failed with 403");
  });
});

describe("admin config and diagnostics", () => {
  it("loads provider config", async () => {
    const config = await fetchAdminConfig({
      baseURL: "http://127.0.0.1:9191",
      token: "admin-token",
      fetchImpl: async () => ({
        ok: true,
        status: 200,
        json: async () => ({ target_base_url: "http://127.0.0.1:11434" })
      })
    });

    expect(config.target_base_url).toBe("http://127.0.0.1:11434");
  });

  it("validates provider target before saving config", async () => {
    await expect(updateAdminConfig({
      baseURL: "http://127.0.0.1:9191",
      token: "admin-token",
      fetchImpl: async () => {
        throw new Error("fetch should not run");
      }
    }, "file:///tmp/provider")).rejects.toThrow("provider target must be a valid http or https URL");
  });

  it("loads diagnostics", async () => {
    const diagnostics = await fetchAdminDiagnostics({
      baseURL: "http://127.0.0.1:9191",
      token: "admin-token",
      fetchImpl: async () => ({
        ok: true,
        status: 200,
        json: async () => ({ health: "ok", recent_failure_categories: { upstream: 2 } })
      })
    });

    expect(diagnostics.recent_failure_categories?.upstream).toBe(2);
  });
});
