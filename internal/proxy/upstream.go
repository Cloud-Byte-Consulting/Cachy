package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloud-byte-consulting/cachy/internal/observability"
)

func transparentProxy(target *url.URL, client *http.Client, logger *slog.Logger, telemetry observability.TelemetryRecorder) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		upstreamURL := upstreamURL(target, r)

		req, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL.String(), r.Body)
		if err != nil {
			observability.LogRequest(r.Context(), logger, observability.RequestLog{
				Request:   r,
				Status:    http.StatusBadGateway,
				Latency:   time.Since(started),
				ErrorKind: observability.ErrorKindProxyError,
			})
			recordTelemetry(telemetry, r, http.StatusBadGateway, time.Since(started), started, observability.ErrorKindProxyError)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		copyRequestHeaders(req.Header, r.Header)
		req.Host = upstreamURL.Host

		resp, err := client.Do(req)
		if err != nil {
			observability.LogRequest(r.Context(), logger, observability.RequestLog{
				Request:   r,
				Status:    http.StatusBadGateway,
				Latency:   time.Since(started),
				ErrorKind: observability.ErrorKindProxyError,
			})
			recordTelemetry(telemetry, r, http.StatusBadGateway, time.Since(started), started, observability.ErrorKindProxyError)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		copyHeaders(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		_, copyErr := copyResponseBody(w, resp)
		observability.LogRequest(r.Context(), logger, observability.RequestLog{
			Request:   r,
			Status:    resp.StatusCode,
			Latency:   time.Since(started),
			ErrorKind: observability.ClassifyProxyError(resp.StatusCode, copyErr),
		})
		recordTelemetry(telemetry, r, resp.StatusCode, time.Since(started), started, observability.ClassifyProxyError(resp.StatusCode, copyErr))
	})
}

func recordTelemetry(telemetry observability.TelemetryRecorder, r *http.Request, status int, latency time.Duration, startedAt time.Time, errKind observability.ErrorKind) {
	if telemetry == nil {
		return
	}
	telemetry.Record(observability.BuildRequestTelemetry(r, status, latency, startedAt, errKind))
}

func upstreamURL(target *url.URL, r *http.Request) url.URL {
	upstreamURL := *target
	upstreamURL.Path = joinPath(target.Path, r.URL.Path)
	upstreamURL.RawQuery = r.URL.RawQuery
	return upstreamURL
}

func copyResponseBody(w http.ResponseWriter, resp *http.Response) (int64, error) {
	if strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		return io.Copy(flushWriter{writer: w}, resp.Body)
	}
	return io.Copy(w, resp.Body)
}

type flushWriter struct {
	writer http.ResponseWriter
}

func (w flushWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if flusher, ok := w.writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return n, err
}
