package ccr

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloud-byte-consulting/cachy/internal/observability"
)

func TestOriginalContentRecoverableOnlyThroughCCRAddress(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	original := []byte("sensitive original block with private diagnostic details")
	address, err := store.Put(original)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	marker, err := RenderMarker(address)
	if err != nil {
		t.Fatalf("RenderMarker() error = %v", err)
	}
	parsed, err := ParseMarker(marker)
	if err != nil {
		t.Fatalf("ParseMarker() error = %v", err)
	}

	recovered, err := store.Get(parsed.Address)
	if err != nil {
		t.Fatalf("Get(parsed address) error = %v", err)
	}
	if string(recovered) != string(original) {
		t.Fatalf("recovered = %q, want original", recovered)
	}

	unrelated := AddressForContent([]byte("different content"))
	if _, err := store.Get(unrelated); err == nil {
		t.Fatal("Get(unrelated address) error = nil, want missing content")
	}
}

func TestCCRMarkerAndDiagnosticsDoNotExposeOriginalContent(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	secret := "private stack trace line with customer-id-123"
	address, err := store.Put([]byte(secret))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	marker, err := RenderMarker(address)
	if err != nil {
		t.Fatalf("RenderMarker() error = %v", err)
	}
	diagnostics, err := store.Diagnostics()
	if err != nil {
		t.Fatalf("Diagnostics() error = %v", err)
	}
	diagnosticsJSON := marshalJSON(t, diagnostics)

	for name, surface := range map[string]string{
		"marker":      marker,
		"diagnostics": diagnosticsJSON,
	} {
		if strings.Contains(surface, secret) || strings.Contains(surface, "customer-id-123") {
			t.Fatalf("%s leaked original content: %s", name, surface)
		}
	}
}

func TestCCRCleanupResultDoesNotExposeOriginalContent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 16, 20, 0, 0, 0, time.UTC)
	store := newTestStore(t)
	secret := "private original that cleanup must not report"
	putWithModTime(t, store, []byte(secret), now.Add(-2*time.Hour))

	result, err := store.Cleanup(CleanupOptions{
		Now:     now,
		Policy:  RetentionPolicy{MaxAge: time.Hour},
		Confirm: false,
	})
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	resultJSON := marshalJSON(t, result)
	if strings.Contains(resultJSON, secret) {
		t.Fatalf("cleanup result leaked original content: %s", resultJSON)
	}
}

func TestCCRLogSurfaceOmitsOriginalContentByDefault(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	secret := "private original prompt content"
	address, err := store.Put([]byte(secret))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	marker, err := RenderMarker(address)
	if err != nil {
		t.Fatalf("RenderMarker() error = %v", err)
	}

	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{ReplaceAttr: observability.RedactAttr}))
	req, err := http.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("X-Prompt", secret)

	observability.LogRequest(context.Background(), logger, observability.RequestLog{
		Request:   req,
		Status:    http.StatusOK,
		Latency:   time.Millisecond,
		ErrorKind: observability.ErrorKindNone,
	})

	line := out.String()
	if strings.Contains(line, secret) {
		t.Fatalf("request log leaked original content: %s", line)
	}
	if strings.Contains(marker, secret) {
		t.Fatalf("marker leaked original content: %s", marker)
	}
}

func marshalJSON(t *testing.T, value any) string {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return string(data)
}
