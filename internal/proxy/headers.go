package proxy

import (
	"net/http"
	"strings"
)

const redactedHeaderValue = "<redacted>"

func copyRequestHeaders(dst, src http.Header) {
	copyHeadersWithPolicy(dst, src, safeRequestHeader)
}

func copyHeaders(dst, src http.Header) {
	copyHeadersWithPolicy(dst, src, func(key string) bool {
		return !hopByHopHeader(key)
	})
}

func copyHeadersWithPolicy(dst, src http.Header, include func(string) bool) {
	for key, values := range src {
		if !include(key) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func safeRequestHeader(key string) bool {
	return !hopByHopHeader(key) && !proxyProvenanceHeader(key) && !requestTransportHeader(key)
}

func hopByHopHeader(key string) bool {
	switch strings.ToLower(key) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
		"te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

func proxyProvenanceHeader(key string) bool {
	switch strings.ToLower(key) {
	case "forwarded", "via", "x-forwarded-for", "x-forwarded-host", "x-forwarded-proto",
		"x-real-ip":
		return true
	default:
		return false
	}
}

func requestTransportHeader(key string) bool {
	switch strings.ToLower(key) {
	case "content-length":
		return true
	default:
		return false
	}
}

func sensitiveHeader(key string) bool {
	switch strings.ToLower(key) {
	case "authorization", "proxy-authorization", "x-api-key", "api-key", "cookie", "set-cookie":
		return true
	default:
		return false
	}
}

func redactHeaderValuesForLog(key string, values []string) []string {
	if !sensitiveHeader(key) {
		return append([]string(nil), values...)
	}

	redacted := make([]string, len(values))
	for i := range redacted {
		redacted[i] = redactedHeaderValue
	}
	return redacted
}
