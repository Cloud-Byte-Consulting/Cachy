package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerRejectsMissingAndWrongTokensWithoutLeakingDetails(t *testing.T) {
	t.Parallel()

	handler, err := New(Config{
		ListenAddress: "127.0.0.1:0",
		Token:         "correct-token",
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{name: "missing"},
		{name: "wrong", token: "wrong-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/admin/v1/status", nil)
			if tt.token != "" {
				req.Header.Set(TokenHeader, tt.token)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			body := rec.Body.String()
			for _, secret := range []string{"correct-token", "wrong-token", TokenHeader} {
				if strings.Contains(body, secret) {
					t.Fatalf("unauthorized body leaked %q: %q", secret, body)
				}
			}
		})
	}
}

func TestHandlerAcceptsAdminTokenHeaders(t *testing.T) {
	t.Parallel()

	handler, err := New(Config{
		ListenAddress: "127.0.0.1:0",
		Token:         "correct-token",
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name      string
		setHeader func(*http.Request)
	}{
		{
			name: "custom header",
			setHeader: func(req *http.Request) {
				req.Header.Set(TokenHeader, "correct-token")
			},
		},
		{
			name: "bearer authorization",
			setHeader: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer correct-token")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/admin/v1/status", nil)
			tt.setHeader(req)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Body.String(); got != "ok" {
				t.Fatalf("body = %q, want ok", got)
			}
		})
	}
}

func TestNewRequiresLocalhostListenAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{name: "ipv4 loopback", address: "127.0.0.1:0"},
		{name: "ipv6 loopback", address: "[::1]:0"},
		{name: "localhost", address: "localhost:0"},
		{name: "all interfaces", address: "0.0.0.0:0", wantErr: true},
		{name: "empty host binds all interfaces", address: ":0", wantErr: true},
		{name: "lan address", address: "192.168.1.20:0", wantErr: true},
		{name: "missing port", address: "127.0.0.1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(Config{
				ListenAddress: tt.address,
				Token:         "correct-token",
			}, http.NotFoundHandler())
			if (err != nil) != tt.wantErr {
				t.Fatalf("New(%q) error = %v, wantErr %v", tt.address, err, tt.wantErr)
			}
		})
	}
}

func TestNewRequiresToken(t *testing.T) {
	t.Parallel()

	_, err := New(Config{ListenAddress: "127.0.0.1:0"}, http.NotFoundHandler())
	if err == nil {
		t.Fatal("New() error = nil, want token validation error")
	}
}

func TestGenerateTokenReturnsOpaqueToken(t *testing.T) {
	t.Parallel()

	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if len(token) < 32 {
		t.Fatalf("token length = %d, want at least 32", len(token))
	}
	if strings.ContainsAny(token, "+/=") {
		t.Fatalf("token %q is not raw URL-safe base64", token)
	}
}
