package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/observability"
)

func TestNewRejectsMissingSchemeOrHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
	}{
		{name: "empty", url: ""},
		{name: "host without scheme", url: "example.com"},
		{name: "scheme without host", url: "https://"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := New(Config{TargetBaseURL: tt.url}); err == nil {
				t.Fatalf("New(%q) error = nil, want error", tt.url)
			}
		})
	}
}

func TestHealthz(t *testing.T) {
	t.Parallel()

	handler, err := New(Config{TargetBaseURL: "http://upstream.example"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("content-type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("content-type = %q, want text/plain; charset=utf-8", got)
	}
	if got := rec.Body.String(); got != "ok\n" {
		t.Fatalf("body = %q, want ok newline", got)
	}
}

func TestTransparentProxyForwardsRequestAndResponse(t *testing.T) {
	t.Parallel()

	var upstreamHost string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/chat/completions" {
			t.Errorf("path = %q, want /api/v1/chat/completions", r.URL.Path)
		}
		if r.URL.RawQuery != "stream=true" {
			t.Errorf("query = %q, want stream=true", r.URL.RawQuery)
		}
		if r.Host != upstreamHost {
			t.Errorf("host = %q, want upstream host %q", r.Host, upstreamHost)
		}
		if got := r.Header.Get("X-Cachy-Test"); got != "present" {
			t.Errorf("forwarded header = %q, want present", got)
		}
		if got := r.Header.Get("Connection"); got != "" {
			t.Errorf("hop-by-hop request header forwarded = %q, want empty", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll request body error = %v", err)
		}
		if got := string(body); got != `{"message":"hello"}` {
			t.Errorf("body = %q, want JSON payload", got)
		}

		w.Header().Set("X-Upstream-Test", "ok")
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("upstream response"))
	}))
	t.Cleanup(upstream.Close)
	upstreamHost = strings.TrimPrefix(upstream.URL, "http://")

	handler, err := New(Config{TargetBaseURL: upstream.URL + "/api"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?stream=true", strings.NewReader(`{"message":"hello"}`))
	req.Header.Set("X-Cachy-Test", "present")
	req.Header.Set("Connection", "keep-alive")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Header().Get("X-Upstream-Test"); got != "ok" {
		t.Fatalf("response header = %q, want ok", got)
	}
	if got := rec.Header().Get("Connection"); got != "" {
		t.Fatalf("hop-by-hop response header = %q, want empty", got)
	}
	if got := rec.Body.String(); got != "upstream response" {
		t.Fatalf("response body = %q, want upstream response", got)
	}
}

func TestTransparentProxyRecordsTelemetryWhenEnabled(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(upstream.Close)

	telemetry := observability.NewMemoryTelemetryStore()
	handler, err := New(Config{
		TargetBaseURL: upstream.URL,
		Telemetry:     telemetry,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/messages?stream=true", strings.NewReader(`{"prompt":"secret prompt body"}`))
	req.Header.Set("Authorization", "Bearer provider-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	records := telemetry.Snapshot()
	if len(records) != 1 {
		t.Fatalf("telemetry records = %d, want 1", len(records))
	}
	got := records[0]
	if got.Method != http.MethodPost || got.Route != "/v1/messages" || !got.QueryPresent {
		t.Fatalf("telemetry request metadata = %#v", got)
	}
	if got.Status != http.StatusAccepted || got.ErrorKind != observability.ErrorKindNone {
		t.Fatalf("telemetry response metadata = %#v", got)
	}
	serialized := stringifyTelemetryRecord(t, got)
	for _, secret := range []string{"secret prompt body", "provider-token", "Authorization"} {
		if strings.Contains(serialized, secret) {
			t.Fatalf("telemetry leaked %q in %s", secret, serialized)
		}
	}
}

func TestHopByHopHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		header string
		want   bool
	}{
		{header: "Connection", want: true},
		{header: "Keep-Alive", want: true},
		{header: "Proxy-Authenticate", want: true},
		{header: "Proxy-Authorization", want: true},
		{header: "TE", want: true},
		{header: "Trailer", want: true},
		{header: "Transfer-Encoding", want: true},
		{header: "Upgrade", want: true},
		{header: "Authorization", want: false},
		{header: "Content-Type", want: false},
		{header: "X-Cachy-Test", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			t.Parallel()

			if got := hopByHopHeader(tt.header); got != tt.want {
				t.Fatalf("hopByHopHeader(%q) = %v, want %v", tt.header, got, tt.want)
			}
		})
	}
}

func TestCopyRequestHeadersAppliesProviderSafePolicy(t *testing.T) {
	t.Parallel()

	src := http.Header{}
	src.Set("Authorization", "Bearer provider-token")
	src.Set("X-Api-Key", "provider-key")
	src.Set("Anthropic-Version", "2023-06-01")
	src.Set("OpenAI-Organization", "org_123")
	src.Set("Content-Type", "application/json")
	src.Set("Accept", "application/json")
	src.Set("X-Custom-Provider-Header", "preserve-me")
	src.Set("Connection", "keep-alive")
	src.Set("Proxy-Authorization", "Basic proxy-secret")
	src.Set("Forwarded", "for=192.0.2.10")
	src.Set("X-Forwarded-For", "192.0.2.10")
	src.Set("X-Real-IP", "192.0.2.10")
	src.Set("Via", "1.1 cachy")
	src.Set("Content-Length", "999")

	dst := http.Header{}
	copyRequestHeaders(dst, src)

	for _, header := range []string{
		"Authorization",
		"X-Api-Key",
		"Anthropic-Version",
		"OpenAI-Organization",
		"Content-Type",
		"Accept",
		"X-Custom-Provider-Header",
	} {
		if got := dst.Get(header); got == "" {
			t.Fatalf("%s was not forwarded", header)
		}
	}

	for _, header := range []string{
		"Connection",
		"Proxy-Authorization",
		"Forwarded",
		"X-Forwarded-For",
		"X-Real-IP",
		"Via",
		"Content-Length",
	} {
		if got := dst.Get(header); got != "" {
			t.Fatalf("%s copied = %q, want empty", header, got)
		}
	}
}

func TestCopyHeadersSkipsHopByHopHeaders(t *testing.T) {
	t.Parallel()

	src := http.Header{}
	src.Add("X-Cachy-Test", "one")
	src.Add("X-Cachy-Test", "two")
	src.Set("Connection", "keep-alive")
	src.Set("Transfer-Encoding", "chunked")

	dst := http.Header{}
	copyHeaders(dst, src)

	if got := dst.Values("X-Cachy-Test"); len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("X-Cachy-Test values = %#v, want [one two]", got)
	}
	if got := dst.Get("Connection"); got != "" {
		t.Fatalf("Connection copied = %q, want empty", got)
	}
	if got := dst.Get("Transfer-Encoding"); got != "" {
		t.Fatalf("Transfer-Encoding copied = %q, want empty", got)
	}
}

func TestRedactHeaderValuesForLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
		values []string
		want   []string
	}{
		{
			name:   "authorization",
			header: "Authorization",
			values: []string{"Bearer provider-token"},
			want:   []string{"<redacted>"},
		},
		{
			name:   "api key",
			header: "X-Api-Key",
			values: []string{"provider-key"},
			want:   []string{"<redacted>"},
		},
		{
			name:   "cookie",
			header: "Cookie",
			values: []string{"session=secret"},
			want:   []string{"<redacted>"},
		},
		{
			name:   "ordinary header",
			header: "Anthropic-Version",
			values: []string{"2023-06-01"},
			want:   []string{"2023-06-01"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := redactHeaderValuesForLog(tt.header, tt.values)
			if len(got) != len(tt.want) {
				t.Fatalf("redacted values = %#v, want %#v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("redacted values = %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		basePath    string
		requestPath string
		want        string
	}{
		{name: "empty base", basePath: "", requestPath: "/v1/chat/completions", want: "/v1/chat/completions"},
		{name: "root request", basePath: "/api", requestPath: "/", want: "/api"},
		{name: "empty request", basePath: "/api", requestPath: "", want: "/api"},
		{name: "joins slashes", basePath: "/api/", requestPath: "/v1/chat/completions", want: "/api/v1/chat/completions"},
		{name: "request without leading slash", basePath: "/api", requestPath: "v1/chat/completions", want: "/api/v1/chat/completions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := joinPath(tt.basePath, tt.requestPath); got != tt.want {
				t.Fatalf("joinPath(%q, %q) = %q, want %q", tt.basePath, tt.requestPath, got, tt.want)
			}
		})
	}
}

func stringifyTelemetryRecord(t *testing.T, record observability.RequestTelemetry) string {
	t.Helper()

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Marshal telemetry error = %v", err)
	}
	return string(data)
}
