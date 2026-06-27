import { useEffect, useMemo, useState } from "react";
import { Activity, AlertTriangle, Gauge, PlugZap, ShieldAlert } from "lucide-react";
import type { DashboardState } from "./status";
import { createDisabledState, createLoadingState, dashboardStateFromRuntime } from "./status";
import type { DesktopAPI, DiagnosticsView, IntegrationKind, ProviderConfigView } from "./desktopApi";
import { getDesktopAPI } from "./desktopApi";
import "./styles.css";

type AppProps = {
  initialState?: DashboardState;
  api?: DesktopAPI;
};

export function App({ initialState, api }: AppProps) {
  const desktopAPI = useMemo(() => api ?? getDesktopAPI(), [api]);
  const [dashboard, setDashboard] = useState<DashboardState>(initialState ?? createLoadingState());

  useEffect(() => {
    if (initialState) {
      return;
    }
    if (!desktopAPI) {
      setDashboard(createDisabledState());
      return;
    }
    let active = true;
    void desktopAPI.getProxyStatus()
      .then((runtime) => {
        if (active) {
          setDashboard(dashboardStateFromRuntime(runtime));
        }
      })
      .catch((error: unknown) => {
        if (!active) {
          return;
        }
        const message = error instanceof Error ? error.message : "Unable to load Cachy status";
        setDashboard(dashboardStateFromRuntime({ state: "error", lastError: message }));
      });
    return () => {
      active = false;
    };
  }, [desktopAPI, initialState]);

  const { status } = dashboard;

  async function handlePrimaryAction() {
    if (!desktopAPI) {
      setDashboard(createDisabledState());
      return;
    }
    setDashboard(createLoadingState());
    try {
      const runtime = status.proxyStatus === "running"
        ? await desktopAPI.stopProxy()
        : await desktopAPI.startProxy();
      setDashboard(dashboardStateFromRuntime(runtime));
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unable to update Cachy proxy";
      setDashboard(dashboardStateFromRuntime({ state: "error", lastError: message }));
    }
  }

  return (
    <main className="shell" aria-label="Cachy desktop companion">
      <aside className="sidebar">
        <div>
          <p className="eyebrow">Cachy</p>
          <h1>Desktop Companion</h1>
        </div>
        <nav aria-label="Primary">
          <a href="#dashboard" aria-current="page">Dashboard</a>
          <a href="#providers">Providers</a>
          <a href="#integrations">Integrations</a>
          <a href="#diagnostics">Diagnostics</a>
        </nav>
      </aside>

      <section id="dashboard" className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">Operational Shell</p>
            <h2>Dashboard</h2>
          </div>
          <span className={`status status-${status.proxyStatus}`}>{status.proxyStatus}</span>
        </header>

        <StateBanner dashboard={dashboard} />

        <div className="metrics" aria-label="Proxy overview">
          <article>
            <Activity aria-hidden="true" />
            <span>Proxy</span>
            <strong>{status.proxyStatus}</strong>
          </article>
          <article>
            <PlugZap aria-hidden="true" />
            <span>Provider</span>
            <strong>{status.activeProvider}</strong>
          </article>
          <article>
            <Gauge aria-hidden="true" />
            <span>Savings</span>
            <strong>{status.savingsToday}</strong>
          </article>
          <article>
            <AlertTriangle aria-hidden="true" />
            <span>Failures</span>
            <strong>{status.recentFailures.length}</strong>
          </article>
        </div>

        <section className="panel" aria-labelledby="next-action-title">
          <div>
            <p className="eyebrow">Next Action</p>
            <h3 id="next-action-title">{status.nextAction}</h3>
          </div>
          <button type="button" onClick={() => void handlePrimaryAction()} disabled={dashboard.kind === "loading"}>
            {status.proxyStatus === "running" ? "Stop Proxy" : "Start Proxy"}
          </button>
        </section>

        <OperationalViews api={desktopAPI} />
      </section>
    </main>
  );
}

function StateBanner({ dashboard }: { dashboard: DashboardState }) {
  if (dashboard.kind === "healthy" || dashboard.kind === "empty") {
    return null;
  }
  return (
    <section className={`banner banner-${dashboard.kind}`} role="status">
      <ShieldAlert aria-hidden="true" />
      <span>{dashboard.message}</span>
    </section>
  );
}

function OperationalViews({ api }: { api?: DesktopAPI }) {
  return (
    <div className="views" aria-label="Operational views">
      <ProviderView api={api} />
      <IntegrationsView api={api} />
      <DiagnosticsView api={api} />
    </div>
  );
}

function ProviderView({ api }: { api?: DesktopAPI }) {
  const [target, setTarget] = useState("");
  const [message, setMessage] = useState("Provider configuration not loaded");

  useEffect(() => {
    if (!api?.getProviderConfig) {
      setMessage("Provider admin API unavailable");
      return;
    }
    let active = true;
    void api.getProviderConfig()
      .then((config: ProviderConfigView) => {
        if (!active) {
          return;
        }
        setTarget(config.targetBaseURL ?? "");
        setMessage(`Credential: ${credentialLabel(config.credentialState)}`);
      })
      .catch((error: unknown) => {
        if (active) {
          setMessage(error instanceof Error ? error.message : "Provider config unavailable");
        }
      });
    return () => {
      active = false;
    };
  }, [api]);

  async function saveProvider() {
    if (!api?.saveProviderTarget) {
      setMessage("Provider admin API unavailable");
      return;
    }
    if (!isHTTPURL(target)) {
      setMessage("Provider target must be a valid http or https URL");
      return;
    }
    try {
      const config = await api.saveProviderTarget(target);
      setTarget(config.targetBaseURL ?? target);
      setMessage("Provider target saved");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Provider target save failed");
    }
  }

  return (
    <section className="tool-panel" id="providers" aria-labelledby="providers-title">
      <div>
        <p className="eyebrow">Providers</p>
        <h3 id="providers-title">Provider Target</h3>
      </div>
      <label>
        Target URL
        <input value={target} onChange={(event) => setTarget(event.target.value)} placeholder="http://127.0.0.1:11434" />
      </label>
      <button type="button" onClick={() => void saveProvider()}>Save Provider</button>
      <p className="detail">{message}</p>
    </section>
  );
}

function IntegrationsView({ api }: { api?: DesktopAPI }) {
  const [output, setOutput] = useState("Select an integration dry run");

  async function runDryRun(kind: IntegrationKind) {
    if (!api?.runIntegrationDryRun) {
      setOutput("Integration dry-run API unavailable");
      return;
    }
    try {
      setOutput(await api.runIntegrationDryRun(kind));
    } catch (error) {
      setOutput(error instanceof Error ? error.message : "Integration dry run failed");
    }
  }

  return (
    <section className="tool-panel" id="integrations" aria-labelledby="integrations-title">
      <div>
        <p className="eyebrow">Integrations</p>
        <h3 id="integrations-title">Dry Runs</h3>
      </div>
      <div className="button-row">
        <button type="button" onClick={() => void runDryRun("codex")}>Codex</button>
        <button type="button" onClick={() => void runDryRun("claude")}>Claude</button>
        <button type="button" onClick={() => void runDryRun("mcp")}>MCP</button>
      </div>
      <pre>{output}</pre>
    </section>
  );
}

function DiagnosticsView({ api }: { api?: DesktopAPI }) {
  const [diagnostics, setDiagnostics] = useState<DiagnosticsView | undefined>();
  const [message, setMessage] = useState("Diagnostics not loaded");

  async function loadDiagnostics() {
    if (!api?.getDiagnostics) {
      setMessage("Diagnostics admin API unavailable");
      return;
    }
    try {
      const next = await api.getDiagnostics();
      setDiagnostics(next);
      setMessage("Diagnostics loaded");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Diagnostics unavailable");
    }
  }

  return (
    <section className="tool-panel" id="diagnostics" aria-labelledby="diagnostics-title">
      <div>
        <p className="eyebrow">Diagnostics</p>
        <h3 id="diagnostics-title">Local Health</h3>
      </div>
      <button type="button" onClick={() => void loadDiagnostics()}>Run Diagnostics</button>
      <dl>
        <dt>Health</dt>
        <dd>{diagnostics?.health ?? "unknown"}</dd>
        <dt>Listen</dt>
        <dd>{diagnostics?.proxyListenAddress ?? "unknown"}</dd>
        <dt>Recent failures</dt>
        <dd>{failureSummary(diagnostics?.recentFailureCategories)}</dd>
      </dl>
      <p className="detail">{message}</p>
    </section>
  );
}

function isHTTPURL(value: string) {
  try {
    const parsed = new URL(value);
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
}

function failureSummary(categories?: Record<string, number>) {
  if (!categories || Object.keys(categories).length === 0) {
    return "none";
  }
  return Object.entries(categories).map(([name, count]) => `${name}: ${count}`).join(", ");
}

function credentialLabel(value?: string) {
  if (!value) {
    return "not configured";
  }
  if (["<redacted>", "present", "missing", "not configured"].includes(value)) {
    return value;
  }
  return "present";
}
