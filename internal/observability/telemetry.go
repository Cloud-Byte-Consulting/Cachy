package observability

import (
	"net/http"
	"sync"
	"time"
)

type TelemetryRecorder interface {
	Record(RequestTelemetry)
}

type RequestTelemetry struct {
	RequestID    string        `json:"request_id,omitempty"`
	SessionID    string        `json:"session_id,omitempty"`
	Provider     string        `json:"provider,omitempty"`
	Method       string        `json:"method,omitempty"`
	Route        string        `json:"route,omitempty"`
	QueryPresent bool          `json:"query_present"`
	Status       int           `json:"status"`
	Latency      time.Duration `json:"latency"`
	ErrorKind    ErrorKind     `json:"error_kind,omitempty"`
	InputTokens  int           `json:"input_tokens,omitempty"`
	OutputTokens int           `json:"output_tokens,omitempty"`
	StartedAt    time.Time     `json:"started_at,omitempty"`

	PromptPreview  string `json:"-"`
	CredentialHint string `json:"-"`
}

type MemoryTelemetryStore struct {
	records []RequestTelemetry
	policy  RetentionPolicy
	mu      sync.RWMutex
}

type RetentionPolicy struct {
	MaxRecords int
	MaxAge     time.Duration
}

type RetentionOption func(*MemoryTelemetryStore)

type CleanupResult struct {
	DryRun      bool `json:"dry_run"`
	WouldRemove int  `json:"would_remove"`
	Removed     int  `json:"removed"`
	Kept        int  `json:"kept"`
}

func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		MaxRecords: 1000,
		MaxAge:     24 * time.Hour,
	}
}

func WithRetentionPolicy(policy RetentionPolicy) RetentionOption {
	return func(s *MemoryTelemetryStore) {
		s.policy = normalizeRetentionPolicy(policy)
	}
}

func NewMemoryTelemetryStore(options ...RetentionOption) *MemoryTelemetryStore {
	store := &MemoryTelemetryStore{policy: DefaultRetentionPolicy()}
	for _, option := range options {
		if option != nil {
			option(store)
		}
	}
	return store
}

func (s *MemoryTelemetryStore) Record(record RequestTelemetry) {
	if s == nil {
		return
	}

	record.PromptPreview = ""
	record.CredentialHint = ""

	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, record)
}

func (s *MemoryTelemetryStore) Snapshot() []RequestTelemetry {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]RequestTelemetry(nil), s.records...)
}

func (s *MemoryTelemetryStore) RetentionPolicy() RetentionPolicy {
	if s == nil {
		return DefaultRetentionPolicy()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.policy
}

func (s *MemoryTelemetryStore) Cleanup(now time.Time, confirm bool) CleanupResult {
	if s == nil {
		return CleanupResult{DryRun: !confirm}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	kept := applyRetentionPolicy(s.records, s.policy, now)
	removed := len(s.records) - len(kept)
	result := CleanupResult{
		DryRun:      !confirm,
		WouldRemove: removed,
		Kept:        len(kept),
	}
	if confirm {
		result.Removed = removed
		s.records = kept
	}
	return result
}

func applyRetentionPolicy(records []RequestTelemetry, policy RetentionPolicy, now time.Time) []RequestTelemetry {
	policy = normalizeRetentionPolicy(policy)
	cutoff := now.Add(-policy.MaxAge)
	kept := make([]RequestTelemetry, 0, len(records))
	for _, record := range records {
		if policy.MaxAge > 0 && !record.StartedAt.IsZero() && record.StartedAt.Before(cutoff) {
			continue
		}
		kept = append(kept, record)
	}
	if policy.MaxRecords > 0 && len(kept) > policy.MaxRecords {
		kept = kept[len(kept)-policy.MaxRecords:]
	}
	return kept
}

func normalizeRetentionPolicy(policy RetentionPolicy) RetentionPolicy {
	defaults := DefaultRetentionPolicy()
	if policy.MaxRecords <= 0 {
		policy.MaxRecords = defaults.MaxRecords
	}
	if policy.MaxAge <= 0 {
		policy.MaxAge = defaults.MaxAge
	}
	return policy
}

func BuildRequestTelemetry(r *http.Request, status int, latency time.Duration, startedAt time.Time, errKind ErrorKind) RequestTelemetry {
	record := RequestTelemetry{
		Status:    status,
		Latency:   latency,
		StartedAt: startedAt,
		ErrorKind: errKind,
	}
	if r == nil {
		return record
	}

	record.RequestID = firstHeader(r.Header, "X-Request-Id", "X-Request-ID")
	record.SessionID = firstHeader(r.Header, "X-Cachy-Session-Id", "X-Session-Id")
	record.Provider = providerFromPath(r.URL.Path)
	record.Method = r.Method
	record.Route = r.URL.Path
	record.QueryPresent = r.URL.RawQuery != ""
	return record
}

func firstHeader(header http.Header, names ...string) string {
	for _, name := range names {
		if value := header.Get(name); value != "" {
			return value
		}
	}
	return ""
}

func providerFromPath(path string) string {
	switch {
	case path == "/v1/messages":
		return "anthropic"
	case path == "/v1/chat/completions" || path == "/v1/responses":
		return "openai"
	default:
		return ""
	}
}
