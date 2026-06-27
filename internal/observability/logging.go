package observability

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const RedactedValue = "<redacted>"

type ErrorKind string

const (
	ErrorKindNone           ErrorKind = "none"
	ErrorKindProxyError     ErrorKind = "proxy_error"
	ErrorKindUpstreamStatus ErrorKind = "upstream_status"
)

type RequestLog struct {
	Request   *http.Request
	Status    int
	Latency   time.Duration
	ErrorKind ErrorKind
}

func LogRequest(ctx context.Context, logger *slog.Logger, entry RequestLog) {
	if logger == nil || entry.Request == nil {
		return
	}

	logger.InfoContext(ctx, "provider request",
		slog.String("method", entry.Request.Method),
		slog.String("path", entry.Request.URL.Path),
		slog.Bool("query_present", entry.Request.URL.RawQuery != ""),
		slog.Int("status", entry.Status),
		slog.Int64("latency_ms", entry.Latency.Milliseconds()),
		slog.String("error_kind", string(entry.ErrorKind)),
	)
}

func ClassifyProxyError(status int, err error) ErrorKind {
	if err != nil {
		return ErrorKindProxyError
	}
	if status >= 500 {
		return ErrorKindUpstreamStatus
	}
	return ErrorKindNone
}

func RedactAttr(_ []string, attr slog.Attr) slog.Attr {
	if sensitiveKey(attr.Key) {
		return slog.String(attr.Key, RedactedValue)
	}
	return attr
}

func sensitiveKey(key string) bool {
	normalized := strings.ToLower(key)
	return strings.Contains(normalized, "authorization") ||
		strings.Contains(normalized, "api_key") ||
		strings.Contains(normalized, "api-key") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "cookie") ||
		strings.Contains(normalized, "prompt")
}
