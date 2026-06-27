import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { App } from "./App";
import { createDisabledState, createLoadingState, dashboardStateFromRuntime } from "./status";

describe("App", () => {
  it("renders loading state", () => {
    render(<App initialState={createLoadingState()} />);

    expect(screen.getByRole("main", { name: /cachy desktop companion/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Dashboard" })).toBeInTheDocument();
    expect(screen.getAllByText("Checking Cachy status...")).toHaveLength(2);
  });

  it("renders empty state", () => {
    render(<App initialState={dashboardStateFromRuntime({ state: "stopped" })} />);

    expect(screen.getAllByText("not-running")).toHaveLength(2);
    expect(screen.getByText("No provider connected")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Start Proxy" })).toBeInTheDocument();
  });

  it("renders healthy state with provider target", () => {
    render(<App initialState={dashboardStateFromRuntime({
      state: "running",
      admin: { target_base_url: "http://127.0.0.1:11434", status: "running" }
    })} />);

    expect(screen.getAllByText("running")).toHaveLength(2);
    expect(screen.getByText("http://127.0.0.1:11434")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Stop Proxy" })).toBeInTheDocument();
  });

  it("renders error state and recent failure count", () => {
    render(<App initialState={dashboardStateFromRuntime({
      state: "error",
      lastError: "port busy"
    })} />);

    expect(screen.getByText("port busy")).toBeInTheDocument();
    expect(screen.getAllByText("error")).toHaveLength(2);
    expect(screen.getByText("1")).toBeInTheDocument();
  });

  it("renders disabled state when preload API is unavailable", () => {
    render(<App initialState={createDisabledState()} />);

    expect(screen.getByText("Desktop controls unavailable")).toBeInTheDocument();
    expect(screen.getAllByText("disabled")).toHaveLength(2);
  });

  it("renders permission-denied state", () => {
    render(<App initialState={dashboardStateFromRuntime({
      state: "running",
      lastError: "admin status request failed with 401"
    })} />);

    expect(screen.getByText("Admin access denied")).toBeInTheDocument();
    expect(screen.getByText("Reconnect with a valid admin token")).toBeInTheDocument();
  });

  it("loads runtime status from the preload API", async () => {
    render(<App api={{
      getProxyStatus: async () => ({ state: "running", admin: { target_base_url: "http://target.local" } }),
      startProxy: async () => ({ state: "running" }),
      stopProxy: async () => ({ state: "stopped" })
    }} />);

    await waitFor(() => expect(screen.getByText("http://target.local")).toBeInTheDocument());
  });

  it("starts and stops through the preload API", async () => {
    const api = {
      getProxyStatus: async () => ({ state: "stopped" as const }),
      startProxy: async () => ({ state: "running" as const }),
      stopProxy: async () => ({ state: "stopped" as const })
    };
    render(<App api={api} />);

    await waitFor(() => expect(screen.getByRole("button", { name: "Start Proxy" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Start Proxy" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Stop Proxy" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Stop Proxy" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Start Proxy" })).toBeInTheDocument());
  });

  it("validates provider target before saving", async () => {
    const api = {
      getProxyStatus: async () => ({ state: "stopped" as const }),
      startProxy: async () => ({ state: "running" as const }),
      stopProxy: async () => ({ state: "stopped" as const }),
      getProviderConfig: async () => ({ targetBaseURL: "" }),
      saveProviderTarget: async () => {
        throw new Error("save should not run");
      }
    };
    render(<App api={api} />);

    await waitFor(() => expect(screen.getByLabelText("Target URL")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Target URL"), { target: { value: "file:///tmp/provider" } });
    fireEvent.click(screen.getByRole("button", { name: "Save Provider" }));

    expect(screen.getByText("Provider target must be a valid http or https URL")).toBeInTheDocument();
  });

  it("saves provider target through the API", async () => {
    const api = {
      getProxyStatus: async () => ({ state: "stopped" as const }),
      startProxy: async () => ({ state: "running" as const }),
      stopProxy: async () => ({ state: "stopped" as const }),
      getProviderConfig: async () => ({ targetBaseURL: "http://127.0.0.1:11434" }),
      saveProviderTarget: async (targetBaseURL: string) => ({ targetBaseURL })
    };
    render(<App api={api} />);

    await waitFor(() => expect(screen.getByDisplayValue("http://127.0.0.1:11434")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Target URL"), { target: { value: "http://127.0.0.1:1234" } });
    fireEvent.click(screen.getByRole("button", { name: "Save Provider" }));

    await waitFor(() => expect(screen.getByText("Provider target saved")).toBeInTheDocument());
  });

  it("renders integration dry-run output", async () => {
    const api = {
      getProxyStatus: async () => ({ state: "stopped" as const }),
      startProxy: async () => ({ state: "running" as const }),
      stopProxy: async () => ({ state: "stopped" as const }),
      runIntegrationDryRun: async () => "cachy integrations codex repair --dry-run"
    };
    render(<App api={api} />);

    fireEvent.click(screen.getByRole("button", { name: "Codex" }));

    await waitFor(() => expect(screen.getByText("cachy integrations codex repair --dry-run")).toBeInTheDocument());
  });

  it("renders diagnostics and recent failures", async () => {
    const api = {
      getProxyStatus: async () => ({ state: "running" as const }),
      startProxy: async () => ({ state: "running" as const }),
      stopProxy: async () => ({ state: "stopped" as const }),
      getDiagnostics: async () => ({
        health: "ok",
        proxyListenAddress: "127.0.0.1:8787",
        recentFailureCategories: { upstream: 2 }
      })
    };
    render(<App api={api} />);

    fireEvent.click(screen.getByRole("button", { name: "Run Diagnostics" }));

    await waitFor(() => expect(screen.getByText("upstream: 2")).toBeInTheDocument());
    expect(screen.getByText("127.0.0.1:8787")).toBeInTheDocument();
  });

  it("renders admin API failures in operational views", async () => {
    const api = {
      getProxyStatus: async () => ({ state: "running" as const }),
      startProxy: async () => ({ state: "running" as const }),
      stopProxy: async () => ({ state: "stopped" as const }),
      getProviderConfig: async () => {
        throw new Error("admin config request failed with 401");
      },
      getDiagnostics: async () => {
        throw new Error("admin diagnostics request failed with 401");
      }
    };
    render(<App api={api} />);

    await waitFor(() => expect(screen.getByText("admin config request failed with 401")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Run Diagnostics" }));

    await waitFor(() => expect(screen.getByText("admin diagnostics request failed with 401")).toBeInTheDocument());
  });

  it("does not render raw provider credentials from config payloads", async () => {
    const api = {
      getProxyStatus: async () => ({ state: "running" as const }),
      startProxy: async () => ({ state: "running" as const }),
      stopProxy: async () => ({ state: "stopped" as const }),
      getProviderConfig: async () => ({
        targetBaseURL: "http://127.0.0.1:11434",
        credentialState: "sk-secret-provider-key"
      })
    };
    render(<App api={api} />);

    await waitFor(() => expect(screen.getByText("Credential: present")).toBeInTheDocument());
    expect(screen.queryByText("sk-secret-provider-key")).not.toBeInTheDocument();
  });
});
