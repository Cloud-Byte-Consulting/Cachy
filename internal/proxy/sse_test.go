package proxy

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSEPassThroughFlushesEventsBeforeUpstreamCompletes(t *testing.T) {
	t.Parallel()

	const firstEvent = "event: content_block_delta\ndata: {\"delta\":\"first\"}\n\n"
	const finalEvent = "data: [DONE]\n\n"

	upstreamFlushed := make(chan struct{})
	releaseFinalEvent := make(chan struct{})

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("Accept = %q, want text/event-stream", r.Header.Get("Accept"))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(firstEvent))
		w.(http.Flusher).Flush()
		close(upstreamFlushed)

		select {
		case <-releaseFinalEvent:
		case <-r.Context().Done():
			return
		}

		_, _ = w.Write([]byte(finalEvent))
		w.(http.Flusher).Flush()
	}))
	t.Cleanup(upstream.Close)

	handler, err := New(Config{TargetBaseURL: upstream.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	proxyServer := httptest.NewServer(handler)
	t.Cleanup(proxyServer.Close)

	req, err := http.NewRequest(http.MethodPost, proxyServer.URL+"/v1/chat/completions", strings.NewReader(`{"stream":true}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")

	client := proxyServer.Client()
	client.Timeout = 3 * time.Second

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}

	select {
	case <-upstreamFlushed:
	case <-time.After(time.Second):
		t.Fatal("upstream did not flush first event")
	}

	reader := bufio.NewReader(resp.Body)
	firstFrame := readSSEFrameAsync(reader)
	select {
	case got := <-firstFrame:
		if got.err != nil {
			t.Fatalf("read first SSE frame error = %v", got.err)
		}
		if got.frame != firstEvent {
			t.Fatalf("first SSE frame = %q, want %q", got.frame, firstEvent)
		}
	case <-time.After(500 * time.Millisecond):
		close(releaseFinalEvent)
		t.Fatal("first SSE frame was not flushed before upstream completed")
	}

	close(releaseFinalEvent)

	finalFrame, err := readSSEFrame(reader)
	if err != nil {
		t.Fatalf("read final SSE frame error = %v", err)
	}
	if finalFrame != finalEvent {
		t.Fatalf("final SSE frame = %q, want %q", finalFrame, finalEvent)
	}
}

type sseFrameResult struct {
	frame string
	err   error
}

func readSSEFrameAsync(reader *bufio.Reader) <-chan sseFrameResult {
	result := make(chan sseFrameResult, 1)
	go func() {
		frame, err := readSSEFrame(reader)
		result <- sseFrameResult{frame: frame, err: err}
	}()
	return result
}

func readSSEFrame(reader *bufio.Reader) (string, error) {
	var frame strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && frame.Len() > 0 {
				return frame.String(), nil
			}
			return "", err
		}
		frame.WriteString(line)
		if line == "\n" {
			return frame.String(), nil
		}
	}
}
