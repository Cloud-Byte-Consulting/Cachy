package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

const TokenHeader = "X-Cachy-Admin-Token"

type Config struct {
	ListenAddress string
	Token         string
}

func New(config Config, next http.Handler) (http.Handler, error) {
	if err := validateListenAddress(config.ListenAddress); err != nil {
		return nil, err
	}
	if config.Token == "" {
		return nil, errors.New("admin token is required")
	}
	if next == nil {
		next = http.NotFoundHandler()
	}
	return requireToken(config.Token, next), nil
}

func GenerateToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate admin token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func requireToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validToken(r, token) {
			w.Header().Set("content-type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized\n"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validToken(r *http.Request, want string) bool {
	for _, got := range requestTokens(r) {
		if tokenEqual(got, want) {
			return true
		}
	}
	return false
}

func requestTokens(r *http.Request) []string {
	tokens := []string{}
	if token := r.Header.Get(TokenHeader); token != "" {
		tokens = append(tokens, token)
	}
	if token := bearerToken(r.Header.Get("Authorization")); token != "" {
		tokens = append(tokens, token)
	}
	return tokens
}

func bearerToken(value string) string {
	fields := strings.Fields(value)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return ""
	}
	return fields[1]
}

func tokenEqual(got, want string) bool {
	gotHash := sha256.Sum256([]byte(got))
	wantHash := sha256.Sum256([]byte(want))
	return subtle.ConstantTimeCompare(gotHash[:], wantHash[:]) == 1
}

func validateListenAddress(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("admin listen address must include host and port: %w", err)
	}
	if host == "" {
		return errors.New("admin listen address must bind to localhost, not all interfaces")
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("admin listen address must bind to localhost")
	}
	return nil
}
