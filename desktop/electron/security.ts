import type { AdminConfig } from "./adminClient.js";

export type RendererProviderConfig = {
  targetBaseURL?: string;
  credentialState?: string;
};

const safeCredentialStates = new Set(["<redacted>", "present", "missing", "not configured"]);

export function providerConfigForRenderer(config: AdminConfig): RendererProviderConfig {
  return {
    targetBaseURL: config.target_base_url,
    credentialState: sanitizeCredentialState(config.provider_credential)
  };
}

export function sanitizeCredentialState(value: string | undefined) {
  if (!value) {
    return undefined;
  }
  if (safeCredentialStates.has(value)) {
    return value;
  }
  return "present";
}

export function safeAdminErrorMessage(error: unknown) {
  const message = error instanceof Error ? error.message : String(error);
  const statusMatch = message.match(/\b(401|403)\b/);
  if (statusMatch) {
    return `admin request failed with ${statusMatch[1]}`;
  }
  return message
    .replace(/Bearer\s+\S+/gi, "Bearer <redacted>")
    .replace(/(token|api[_-]?key|secret)=\S+/gi, "$1=<redacted>");
}
