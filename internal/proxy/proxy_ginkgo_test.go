package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Proxy", func() {
	Describe("health endpoint", func() {
		It("returns an OK plaintext response", func() {
			handler, err := New(Config{TargetBaseURL: "http://upstream.example"})
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Header().Get("content-type")).To(Equal("text/plain; charset=utf-8"))
			Expect(rec.Body.String()).To(Equal("ok\n"))
		})
	})

	Describe("target validation", func() {
		DescribeTable("rejects targets without a scheme and host",
			func(target string) {
				_, err := New(Config{TargetBaseURL: target})
				Expect(err).To(MatchError(ContainSubstring("target base URL must include scheme and host")))
			},
			Entry("empty", ""),
			Entry("host without scheme", "example.com"),
			Entry("scheme without host", "https://"),
		)
	})

	Describe("path joining", func() {
		It("preserves target base paths while forwarding request paths", func() {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.Path).To(Equal("/api/v1/chat/completions"))
				Expect(r.URL.RawQuery).To(Equal("stream=true"))
				_, _ = w.Write([]byte("ok"))
			}))
			DeferCleanup(upstream.Close)

			handler, err := New(Config{TargetBaseURL: upstream.URL + "/api"})
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?stream=true", strings.NewReader(`{"message":"hello"}`))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(Equal("ok"))
		})
	})
})
