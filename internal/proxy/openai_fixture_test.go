package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIChatCompletionsPassThroughFixture(t *testing.T) {
	t.Parallel()

	requestBody := `{"model":"gpt-4.1-mini","messages":[{"role":"user","content":"hello"}],"stream":false}`
	responseBody := `{"id":"chatcmpl_fixture","object":"chat.completion","choices":[{"message":{"role":"assistant","content":"hi"}}]}`

	handler := openAIFixtureHandler(t, openAIFixture{
		method:          http.MethodPost,
		path:            "/v1/chat/completions",
		query:           "timeout=30",
		requestBody:     requestBody,
		responseStatus:  http.StatusCreated,
		responseBody:    responseBody,
		responseHeader:  "X-OpenAI-Request-ID",
		responseValue:   "req_chat_fixture",
		requiredHeaders: map[string]string{"Authorization": "Bearer openai-token", "Content-Type": "application/json", "OpenAI-Organization": "org_fixture"},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?timeout=30", strings.NewReader(requestBody))
	req.Header.Set("Authorization", "Bearer openai-token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Organization", "org_fixture")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertFixtureResponse(t, rec, http.StatusCreated, "X-OpenAI-Request-ID", "req_chat_fixture", responseBody)
}

func TestOpenAIResponsesPassThroughFixture(t *testing.T) {
	t.Parallel()

	requestBody := `{"model":"gpt-4.1-mini","input":[{"role":"user","content":[{"type":"input_text","text":"summarize this"}]}]}`
	responseBody := `{"id":"resp_fixture","object":"response","output":[{"type":"message","content":[{"type":"output_text","text":"summary"}]}]}`

	handler := openAIFixtureHandler(t, openAIFixture{
		method:          http.MethodPost,
		path:            "/v1/responses",
		requestBody:     requestBody,
		responseStatus:  http.StatusOK,
		responseBody:    responseBody,
		responseHeader:  "X-OpenAI-Request-ID",
		responseValue:   "req_responses_fixture",
		requiredHeaders: map[string]string{"Authorization": "Bearer openai-token", "Content-Type": "application/json", "OpenAI-Project": "proj_fixture"},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(requestBody))
	req.Header.Set("Authorization", "Bearer openai-token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Project", "proj_fixture")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertFixtureResponse(t, rec, http.StatusOK, "X-OpenAI-Request-ID", "req_responses_fixture", responseBody)
}

type openAIFixture struct {
	method          string
	path            string
	query           string
	requestBody     string
	responseStatus  int
	responseBody    string
	responseHeader  string
	responseValue   string
	requiredHeaders map[string]string
}

func openAIFixtureHandler(t *testing.T, fixture openAIFixture) http.Handler {
	t.Helper()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != fixture.method {
			t.Errorf("method = %s, want %s", r.Method, fixture.method)
		}
		if r.URL.Path != fixture.path {
			t.Errorf("path = %q, want %q", r.URL.Path, fixture.path)
		}
		if r.URL.RawQuery != fixture.query {
			t.Errorf("query = %q, want %q", r.URL.RawQuery, fixture.query)
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

		w.Header().Set(fixture.responseHeader, fixture.responseValue)
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

func assertFixtureResponse(t *testing.T, rec *httptest.ResponseRecorder, status int, header, value, body string) {
	t.Helper()

	if rec.Code != status {
		t.Fatalf("status = %d, want %d", rec.Code, status)
	}
	if got := rec.Header().Get(header); got != value {
		t.Fatalf("%s = %q, want %q", header, got, value)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := rec.Body.String(); got != body {
		t.Fatalf("body = %q, want %q", got, body)
	}
}
