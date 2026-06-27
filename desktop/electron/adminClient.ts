export type AdminStatus = {
  status?: string;
  version?: string;
  proxy_listen_address?: string;
  target_base_url?: string;
};

export type AdminConfig = {
  target_base_url?: string;
  provider_credential?: string;
};

export type AdminDiagnostics = {
  health?: string;
  recent_failure_categories?: Record<string, number>;
  proxy_listen_address?: string;
  config_dir?: string;
};

export type FetchLike = (input: string, init?: {
  method?: string;
  headers?: Record<string, string>;
  body?: string;
}) => Promise<{
  ok: boolean;
  status: number;
  json: () => Promise<unknown>;
}>;

export type AdminClientOptions = {
  baseURL: string;
  token: string;
  fetchImpl?: FetchLike;
};

export async function fetchAdminStatus(options: AdminClientOptions): Promise<AdminStatus> {
  const fetchImpl = options.fetchImpl ?? fetch;
  const baseURL = options.baseURL.replace(/\/+$/, "");
  const response = await fetchImpl(`${baseURL}/admin/v1/status`, {
    headers: adminHeaders(options.token)
  });
  if (!response.ok) {
    throw new Error(`admin status request failed with ${response.status}`);
  }
  return response.json() as Promise<AdminStatus>;
}

export async function fetchAdminConfig(options: AdminClientOptions): Promise<AdminConfig> {
  const fetchImpl = options.fetchImpl ?? fetch;
  const baseURL = options.baseURL.replace(/\/+$/, "");
  const response = await fetchImpl(`${baseURL}/admin/v1/config`, {
    headers: adminHeaders(options.token)
  });
  if (!response.ok) {
    throw new Error(`admin config request failed with ${response.status}`);
  }
  return response.json() as Promise<AdminConfig>;
}

export async function updateAdminConfig(options: AdminClientOptions, targetBaseURL: string): Promise<AdminConfig> {
  validateProviderTarget(targetBaseURL);
  const fetchImpl = options.fetchImpl ?? fetch;
  const baseURL = options.baseURL.replace(/\/+$/, "");
  const response = await fetchImpl(`${baseURL}/admin/v1/config`, {
    method: "PUT",
    headers: {
      ...adminHeaders(options.token),
      "Content-Type": "application/json"
    },
    body: JSON.stringify({ target_base_url: targetBaseURL })
  });
  if (!response.ok) {
    throw new Error(`admin config update failed with ${response.status}`);
  }
  return response.json() as Promise<AdminConfig>;
}

export async function fetchAdminDiagnostics(options: AdminClientOptions): Promise<AdminDiagnostics> {
  const fetchImpl = options.fetchImpl ?? fetch;
  const baseURL = options.baseURL.replace(/\/+$/, "");
  const response = await fetchImpl(`${baseURL}/admin/v1/diagnostics`, {
    headers: adminHeaders(options.token)
  });
  if (!response.ok) {
    throw new Error(`admin diagnostics request failed with ${response.status}`);
  }
  return response.json() as Promise<AdminDiagnostics>;
}

export function validateProviderTarget(targetBaseURL: string) {
  try {
    const parsed = new URL(targetBaseURL);
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      throw new Error("provider target must use http or https");
    }
  } catch {
    throw new Error("provider target must be a valid http or https URL");
  }
}

function adminHeaders(token: string) {
  return {
    Authorization: `Bearer ${token}`
  };
}
