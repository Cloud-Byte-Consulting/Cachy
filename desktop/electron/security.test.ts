import { describe, expect, it } from "vitest";
import { providerConfigForRenderer, safeAdminErrorMessage, sanitizeCredentialState } from "./security";

describe("Electron security boundary", () => {
  it("does not expose raw provider credentials in renderer config", () => {
    const config = providerConfigForRenderer({
      target_base_url: "http://127.0.0.1:11434",
      provider_credential: "sk-secret-provider-key"
    });

    expect(config).toEqual({
      targetBaseURL: "http://127.0.0.1:11434",
      credentialState: "present"
    });
    expect(JSON.stringify(config)).not.toContain("sk-secret-provider-key");
  });

  it("preserves known redacted credential states", () => {
    expect(sanitizeCredentialState("<redacted>")).toBe("<redacted>");
    expect(sanitizeCredentialState("missing")).toBe("missing");
  });

  it("does not echo admin tokens in renderer-facing error messages", () => {
    const message = safeAdminErrorMessage(new Error("admin config request failed with 401 token=secret-admin-token"));

    expect(message).toBe("admin request failed with 401");
    expect(message).not.toContain("secret-admin-token");
  });
});
