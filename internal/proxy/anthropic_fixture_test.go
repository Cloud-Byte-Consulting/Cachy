package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnthropicMessagesPassThroughFixture(t *testing.T) {
	t.Parallel()

	requestBody := `{"model":"claude-3-5-sonnet-latest","max_tokens":128,"messages":[{"role":"user","content":"hello"}]}`
	responseBody := `{"id":"msg_fixture","type":"message","role":"assistant","content":[{"type":"text","text":"hi"}],"stop_reason":"end_turn"}`

	handler := anthropicFixtureHandler(t, anthropicFixture{
		method:         http.MethodPost,
		path:           "/v1/messages",
		requestBody:    requestBody,
		responseStatus: http.StatusOK,
		responseBody:   responseBody,
		requiredHeaders: map[string]string{
			"X-Api-Key":         "anthropic-token",
			"Anthropic-Version": "2023-06-01",
			"Anthropic-Beta":    "messages-2023-12-15",
			"Content-Type":      "application/json",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(requestBody))
	req.Header.Set("X-Api-Key", "anthropic-token")
	req.Header.Set("Anthropic-Version", "2023-06-01")
	req.Header.Set("Anthropic-Beta", "messages-2023-12-15")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Request-ID"); got != "req_anthropic_fixture" {
		t.Fatalf("Request-ID = %q, want req_anthropic_fixture", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := rec.Body.String(); got != responseBody {
		t.Fatalf("body = %q, want %q", got, responseBody)
	}
}

type anthropicFixture struct {
	method          string
	path            string
	requestBody     string
	responseStatus  int
	responseBody    string
	requiredHeaders map[string]string
}

func anthropicFixtureHandler(t *testing.T, fixture anthropicFixture) http.Handler {
	t.Helper()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != fixture.method {
			t.Errorf("method = %s, want %s", r.Method, fixture.method)
		}
		if r.URL.Path != fixture.path {
			t.Errorf("path = %q, want %q", r.URL.Path, fixture.path)
		}
		for header, want := range fixture.requiredHeaders {
			if got := r.Header.Get(header); got != want {
				t.Errorf("%s = %q, want %q", header, got, want)
			}
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll request body error = %v", err)
		}
		if got := string(body); got != fixture.requestBody {
			t.Errorf("body = %q, want %q", got, fixture.requestBody)
		}

		w.Header().Set("Request-ID", "req_anthropic_fixture")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(fixture.responseStatus)
		_, _ = w.Write([]byte(fixture.responseBody))
	}))
	t.Cleanup(upstream.Close)

	handler, err := New(Config{TargetBaseURL: upstream.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return handler
}
