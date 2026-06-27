package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestRequestLogOmitsSecretsAndPromptPayload(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{ReplaceAttr: RedactAttr}))

	req, err := http.NewRequest(http.MethodPost, "/v1/chat/completions?stream=true", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Authorization", "Bearer provider-token")
	req.Header.Set("X-Api-Key", "provider-key")
	req.Header.Set("X-Prompt", "full prompt payload")

	LogRequest(context.Background(), logger, RequestLog{
		Request:   req,
		Status:    http.StatusOK,
		Latency:   25 * time.Millisecond,
		ErrorKind: ErrorKindNone,
	})

	line := out.String()
	for _, secret := range []string{"provider-token", "provider-key", "full prompt payload", "Authorization", "X-Api-Key", "X-Prompt"} {
		if bytes.Contains([]byte(line), []byte(secret)) {
			t.Fatalf("log leaked %q in %s", secret, line)
		}
	}

	var record map[string]any
	if err := json.Unmarshal(out.Bytes(), &record); err != nil {
		t.Fatalf("Unmarshal log record error = %v", err)
	}

	assertRecordValue(t, record, "method", http.MethodPost)
	assertRecordValue(t, record, "path", "/v1/chat/completions")
	assertRecordValue(t, record, "query_present", true)
	assertRecordValue(t, record, "status", float64(http.StatusOK))
	assertRecordValue(t, record, "error_kind", string(ErrorKindNone))
	if _, ok := record["latency_ms"]; !ok {
		t.Fatal("latency_ms missing from log record")
	}
}

func TestClassifyProxyError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		err    error
		want   ErrorKind
	}{
		{name: "none", status: http.StatusOK, want: ErrorKindNone},
		{name: "upstream status", status: http.StatusServiceUnavailable, want: ErrorKindUpstreamStatus},
		{name: "proxy error", err: context.Canceled, want: ErrorKindProxyError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ClassifyProxyError(tt.status, tt.err); got != tt.want {
				t.Fatalf("ClassifyProxyError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRedactAttrRedactsSensitiveKeys(t *testing.T) {
	t.Parallel()

	attr := RedactAttr(nil, slog.String("authorization", "Bearer provider-token"))
	if attr.Value.String() != RedactedValue {
		t.Fatalf("redacted attr = %q, want %q", attr.Value.String(), RedactedValue)
	}

	attr = RedactAttr(nil, slog.String("path", "/v1/messages"))
	if attr.Value.String() != "/v1/messages" {
		t.Fatalf("ordinary attr = %q, want preserved", attr.Value.String())
	}
}

func assertRecordValue(t *testing.T, record map[string]any, key string, want any) {
	t.Helper()

	if got := record[key]; got != want {
		t.Fatalf("%s = %#v, want %#v", key, got, want)
	}
}
