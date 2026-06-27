package observability

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMemoryTelemetryStoreRecordsRequestMetadataWithoutSensitiveContent(t *testing.T) {
	t.Parallel()

	store := NewMemoryTelemetryStore()
	started := time.Date(2026, 6, 16, 1, 0, 0, 0, time.UTC)
	store.Record(RequestTelemetry{
		RequestID:      "req_123",
		SessionID:      "session_1",
		Provider:       "openai",
		Method:         http.MethodPost,
		Route:          "/v1/chat/completions",
		QueryPresent:   true,
		Status:         http.StatusOK,
		Latency:        25 * time.Millisecond,
		ErrorKind:      ErrorKindNone,
		InputTokens:    100,
		OutputTokens:   20,
		StartedAt:      started,
		PromptPreview:  "secret prompt body",
		CredentialHint: "Bearer provider-token",
	})

	records := store.Snapshot()
	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}
	got := records[0]
	if got.RequestID != "req_123" || got.SessionID != "session_1" || got.Provider != "openai" {
		t.Fatalf("unexpected record identity: %#v", got)
	}
	if got.Method != http.MethodPost || got.Route != "/v1/chat/completions" || !got.QueryPresent {
		t.Fatalf("unexpected request metadata: %#v", got)
	}
	if got.Status != http.StatusOK || got.Latency != 25*time.Millisecond || got.ErrorKind != ErrorKindNone {
		t.Fatalf("unexpected response metadata: %#v", got)
	}
	if got.InputTokens != 100 || got.OutputTokens != 20 {
		t.Fatalf("token counts = %d/%d, want 100/20", got.InputTokens, got.OutputTokens)
	}
	if !got.StartedAt.Equal(started) {
		t.Fatalf("started_at = %v, want %v", got.StartedAt, started)
	}

	serialized := stringifyTelemetry(t, records)
	for _, secret := range []string{"secret prompt body", "provider-token", "Bearer provider-token"} {
		if strings.Contains(serialized, secret) {
			t.Fatalf("telemetry persisted sensitive value %q in %s", secret, serialized)
		}
	}
}

func TestMemoryTelemetryStoreSnapshotIsStableCopy(t *testing.T) {
	t.Parallel()

	store := NewMemoryTelemetryStore()
	store.Record(RequestTelemetry{RequestID: "first"})
	store.Record(RequestTelemetry{RequestID: "second"})

	records := store.Snapshot()
	records[0].RequestID = "changed"

	again := store.Snapshot()
	if again[0].RequestID != "first" || again[1].RequestID != "second" {
		t.Fatalf("snapshot mutated store records: %#v", again)
	}
}

func TestDefaultRetentionPolicyIsExplicit(t *testing.T) {
	t.Parallel()

	policy := DefaultRetentionPolicy()
	if policy.MaxRecords != 1000 {
		t.Fatalf("default max records = %d, want 1000", policy.MaxRecords)
	}
	if policy.MaxAge != 24*time.Hour {
		t.Fatalf("default max age = %s, want 24h", policy.MaxAge)
	}
}

func TestCleanupDryRunDoesNotRemoveExpiredTelemetry(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 16, 1, 0, 0, 0, time.UTC)
	store := NewMemoryTelemetryStore(WithRetentionPolicy(RetentionPolicy{
		MaxRecords: 10,
		MaxAge:     time.Hour,
	}))
	store.Record(RequestTelemetry{RequestID: "old", StartedAt: now.Add(-2 * time.Hour)})
	store.Record(RequestTelemetry{RequestID: "new", StartedAt: now})

	result := store.Cleanup(now, false)

	if !result.DryRun {
		t.Fatal("cleanup dry run flag = false, want true")
	}
	if result.WouldRemove != 1 || result.Removed != 0 {
		t.Fatalf("cleanup result = %#v, want one would-remove and zero removed", result)
	}
	if got := len(store.Snapshot()); got != 2 {
		t.Fatalf("records after dry run = %d, want 2", got)
	}
}

func TestCleanupConfirmedRemovesExpiredTelemetry(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 16, 1, 0, 0, 0, time.UTC)
	store := NewMemoryTelemetryStore(WithRetentionPolicy(RetentionPolicy{
		MaxRecords: 10,
		MaxAge:     time.Hour,
	}))
	store.Record(RequestTelemetry{RequestID: "old", StartedAt: now.Add(-2 * time.Hour)})
	store.Record(RequestTelemetry{RequestID: "new", StartedAt: now})

	result := store.Cleanup(now, true)

	if result.DryRun {
		t.Fatal("cleanup dry run flag = true, want false")
	}
	if result.Removed != 1 || result.Kept != 1 {
		t.Fatalf("cleanup result = %#v, want one removed and one kept", result)
	}
	records := store.Snapshot()
	if len(records) != 1 || records[0].RequestID != "new" {
		t.Fatalf("records after cleanup = %#v, want only new record", records)
	}
}

func TestCleanupTrimsToMaxRecords(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 16, 1, 0, 0, 0, time.UTC)
	store := NewMemoryTelemetryStore(WithRetentionPolicy(RetentionPolicy{
		MaxRecords: 2,
		MaxAge:     24 * time.Hour,
	}))
	store.Record(RequestTelemetry{RequestID: "first", StartedAt: now})
	store.Record(RequestTelemetry{RequestID: "second", StartedAt: now})
	store.Record(RequestTelemetry{RequestID: "third", StartedAt: now})

	result := store.Cleanup(now, true)

	if result.Removed != 1 || result.Kept != 2 {
		t.Fatalf("cleanup result = %#v, want one trimmed and two kept", result)
	}
	records := store.Snapshot()
	if len(records) != 2 || records[0].RequestID != "second" || records[1].RequestID != "third" {
		t.Fatalf("records after trim = %#v, want second and third", records)
	}
}

func stringifyTelemetry(t *testing.T, records []RequestTelemetry) string {
	t.Helper()

	data, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("Marshal telemetry error = %v", err)
	}
	return string(data)
}
