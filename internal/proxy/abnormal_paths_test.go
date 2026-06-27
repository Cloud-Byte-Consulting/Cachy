package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProxyPassesLargeRequestAndResponseBodies(t *testing.T) {
	t.Parallel()

	requestBody := strings.Repeat("request-body-", 128*1024)
	responseBody := strings.Repeat("response-body-", 128*1024)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll request body error = %v", err)
		}
		if got := string(body); got != requestBody {
			t.Errorf("large request body length = %d, want %d", len(got), len(requestBody))
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, responseBody)
	}))
	t.Cleanup(upstream.Close)

	handler, err := New(Config{TargetBaseURL: upstream.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(requestBody))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if got := rec.Body.String(); got != responseBody {
		t.Fatalf("large response body length = %d, want %d", len(got), len(responseBody))
	}
}

func TestProxyPreservesUpstreamFailureResponse(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Upstream-Error", "rate-limited")
		http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(upstream.Close)

	handler, err := New(Config{TargetBaseURL: upstream.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"message":"hello"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Header().Get("X-Upstream-Error"); got != "rate-limited" {
		t.Fatalf("X-Upstream-Error = %q, want rate-limited", got)
	}
	if got := rec.Body.String(); got != "provider unavailable\n" {
		t.Fatalf("body = %q, want provider unavailable newline", got)
	}
}

func TestProxyReturnsBadGatewayWhenUpstreamCannotBeReached(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	target := upstream.URL
	upstream.Close()

	handler, err := New(Config{TargetBaseURL: target})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"input":"hello"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

func TestProxyPropagatesClientCancellationUpstream(t *testing.T) {
	t.Parallel()

	transport := &cancellationTransport{
		started:  make(chan struct{}),
		canceled: make(chan struct{}),
	}

	handler, err := New(Config{
		TargetBaseURL: "http://upstream.example",
		Client:        &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"message":"hello"}`)).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rec, req)
		close(done)
	}()

	select {
	case <-transport.started:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("upstream transport did not receive request")
	}

	cancel()

	select {
	case <-transport.canceled:
	case <-time.After(time.Second):
		t.Fatal("upstream transport did not observe request cancellation")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("proxy handler did not finish after cancellation")
	}

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

type cancellationTransport struct {
	started  chan struct{}
	canceled chan struct{}
}

func (t *cancellationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	close(t.started)
	<-req.Context().Done()
	close(t.canceled)
	return nil, req.Context().Err()
}
