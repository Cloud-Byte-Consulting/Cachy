package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloud-byte-consulting/cachy/internal/observability"
	"github.com/cloud-byte-consulting/cachy/internal/platform"
)

func TestServerReturnsAuthenticatedStatus(t *testing.T) {
	t.Parallel()

	handler := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/status", nil)
	req.Header.Set(TokenHeader, "admin-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got StatusResponse
	decodeJSON(t, rec, &got)
	if got.Status != "ok" {
		t.Fatalf("status response status = %q, want ok", got.Status)
	}
	if got.Version != "dev" {
		t.Fatalf("version = %q, want dev", got.Version)
	}
	if got.Proxy.ListenAddress != "127.0.0.1:8787" {
		t.Fatalf("proxy listen = %q, want configured listen", got.Proxy.ListenAddress)
	}
	if !got.Proxy.TargetConfigured {
		t.Fatal("target_configured = false, want true")
	}
	if got.Paths.ConfigDir != "/cfg/cachy" {
		t.Fatalf("config dir = %q, want /cfg/cachy", got.Paths.ConfigDir)
	}
}

func TestServerReturnsRedactedConfig(t *testing.T) {
	t.Parallel()

	handler := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/config", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got ConfigResponse
	decodeJSON(t, rec, &got)
	if got.TargetBaseURL != "http://provider.example" {
		t.Fatalf("target = %q, want configured target", got.TargetBaseURL)
	}
	if got.ProviderCredential != Redacted {
		t.Fatalf("provider credential = %q, want redacted", got.ProviderCredential)
	}
	if strings.Contains(rec.Body.String(), "provider-secret") {
		t.Fatalf("config response leaked provider secret: %s", rec.Body.String())
	}
}

func TestServerUpdatesTargetConfig(t *testing.T) {
	t.Parallel()

	handler := newTestServer(t)
	body := bytes.NewBufferString(`{"target_base_url":"http://new-provider.example/v1"}`)
	req := httptest.NewRequest(http.MethodPut, "/admin/v1/config", body)
	req.Header.Set(TokenHeader, "admin-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body %q", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got ConfigResponse
	decodeJSON(t, rec, &got)
	if got.TargetBaseURL != "http://new-provider.example/v1" {
		t.Fatalf("target = %q, want updated target", got.TargetBaseURL)
	}
	if got.ProviderCredential != Redacted {
		t.Fatalf("provider credential = %q, want redacted", got.ProviderCredential)
	}
}

func TestServerRejectsInvalidConfigWrite(t *testing.T) {
	t.Parallel()

	handler := newTestServer(t)
	body := bytes.NewBufferString(`{"target_base_url":"://bad"}`)
	req := httptest.NewRequest(http.MethodPut, "/admin/v1/config", body)
	req.Header.Set(TokenHeader, "admin-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if strings.Contains(rec.Body.String(), "provider-secret") {
		t.Fatalf("error response leaked provider secret: %s", rec.Body.String())
	}
}

func TestServerRequiresAdminToken(t *testing.T) {
	t.Parallel()

	handler := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestServerReturnsMetricsFromTelemetry(t *testing.T) {
	t.Parallel()

	telemetry := observability.NewMemoryTelemetryStore()
	telemetry.Record(observability.RequestTelemetry{
		Status:    http.StatusOK,
		Latency:   20 * time.Millisecond,
		ErrorKind: observability.ErrorKindNone,
	})
	telemetry.Record(observability.RequestTelemetry{
		Status:    http.StatusBadGateway,
		Latency:   40 * time.Millisecond,
		ErrorKind: observability.ErrorKindProxyError,
	})
	handler := newTestServerWithTelemetry(t, telemetry)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/metrics", nil)
	req.Header.Set(TokenHeader, "admin-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got MetricsResponse
	decodeJSON(t, rec, &got)
	if got.RequestsTotal != 2 || got.ErrorsTotal != 1 {
		t.Fatalf("metrics totals = %#v, want 2 requests and 1 error", got)
	}
	if got.AverageLatencyMS != 30 {
		t.Fatalf("average latency = %d, want 30", got.AverageLatencyMS)
	}
	if got.StatusCounts["200"] != 1 || got.StatusCounts["502"] != 1 {
		t.Fatalf("status counts = %#v, want 200 and 502", got.StatusCounts)
	}
	if got.FailureCategories[string(observability.ErrorKindProxyError)] != 1 {
		t.Fatalf("failure categories = %#v, want proxy_error count", got.FailureCategories)
	}
}

func TestServerReturnsDiagnosticsWithPathsPortsAndFailureCategories(t *testing.T) {
	t.Parallel()

	telemetry := observability.NewMemoryTelemetryStore()
	telemetry.Record(observability.RequestTelemetry{
		Status:    http.StatusServiceUnavailable,
		ErrorKind: observability.ErrorKindUpstreamStatus,
	})
	handler := newTestServerWithTelemetry(t, telemetry)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/diagnostics", nil)
	req.Header.Set(TokenHeader, "admin-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got DiagnosticsResponse
	decodeJSON(t, rec, &got)
	if got.Health != "ok" {
		t.Fatalf("health = %q, want ok", got.Health)
	}
	if got.Paths.ConfigDir != "/cfg/cachy" {
		t.Fatalf("config dir = %q, want /cfg/cachy", got.Paths.ConfigDir)
	}
	if got.Ports.ProxyListenAddress != "127.0.0.1:8787" || got.Ports.AdminListenAddress != "127.0.0.1:0" {
		t.Fatalf("ports = %#v, want proxy and admin listen addresses", got.Ports)
	}
	if got.FailureCategories[string(observability.ErrorKindUpstreamStatus)] != 1 {
		t.Fatalf("failure categories = %#v, want upstream_status count", got.FailureCategories)
	}
}

func TestServerReturnsEmptyMetricsWithoutTelemetry(t *testing.T) {
	t.Parallel()

	handler := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/metrics", nil)
	req.Header.Set(TokenHeader, "admin-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got MetricsResponse
	decodeJSON(t, rec, &got)
	if got.RequestsTotal != 0 || got.ErrorsTotal != 0 || got.AverageLatencyMS != 0 {
		t.Fatalf("empty metrics = %#v, want zero values", got)
	}
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()

	return newTestServerWithTelemetry(t, nil)
}

func newTestServerWithTelemetry(t *testing.T, telemetry TelemetrySnapshotter) http.Handler {
	t.Helper()

	handler, err := NewServer(ServerConfig{
		ListenAddress: "127.0.0.1:0",
		Token:         "admin-token",
		Version:       "dev",
		Paths: platform.Paths{
			ConfigDir: "/cfg/cachy",
			StateDir:  "/state/cachy",
			CacheDir:  "/cache/cachy",
		},
		RuntimeConfig: RuntimeConfig{
			ProxyListenAddress: "127.0.0.1:8787",
			TargetBaseURL:      "http://provider.example",
			ProviderCredential: "provider-secret",
		},
		Telemetry: telemetry,
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return handler
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder, target *T) {
	t.Helper()

	if got := rec.Header().Get("content-type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("decode JSON error = %v body %q", err, rec.Body.String())
	}
}
