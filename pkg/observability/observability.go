// Package observability exposes Cachy's privacy-safe telemetry types for
// applications embedding the proxy handler.
package observability

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	internalobservability "github.com/cloud-byte-consulting/cachy/internal/observability"
)

const RedactedValue = internalobservability.RedactedValue

type ErrorKind = internalobservability.ErrorKind

const (
	ErrorKindNone           = internalobservability.ErrorKindNone
	ErrorKindProxyError     = internalobservability.ErrorKindProxyError
	ErrorKindUpstreamStatus = internalobservability.ErrorKindUpstreamStatus
)

type TelemetryRecorder = internalobservability.TelemetryRecorder
type RequestTelemetry = internalobservability.RequestTelemetry
type MemoryTelemetryStore = internalobservability.MemoryTelemetryStore
type RetentionPolicy = internalobservability.RetentionPolicy
type RetentionOption = internalobservability.RetentionOption
type CleanupResult = internalobservability.CleanupResult
type RequestLog = internalobservability.RequestLog

func DefaultRetentionPolicy() RetentionPolicy {
	return internalobservability.DefaultRetentionPolicy()
}

func WithRetentionPolicy(policy RetentionPolicy) RetentionOption {
	return internalobservability.WithRetentionPolicy(policy)
}

func NewMemoryTelemetryStore(options ...RetentionOption) *MemoryTelemetryStore {
	return internalobservability.NewMemoryTelemetryStore(options...)
}

func BuildRequestTelemetry(r *http.Request, status int, latency time.Duration, startedAt time.Time, errKind ErrorKind) RequestTelemetry {
	return internalobservability.BuildRequestTelemetry(r, status, latency, startedAt, errKind)
}

func LogRequest(ctx context.Context, logger *slog.Logger, entry RequestLog) {
	internalobservability.LogRequest(ctx, logger, entry)
}

func ClassifyProxyError(status int, err error) ErrorKind {
	return internalobservability.ClassifyProxyError(status, err)
}

func RedactAttr(groups []string, attr slog.Attr) slog.Attr {
	return internalobservability.RedactAttr(groups, attr)
}
