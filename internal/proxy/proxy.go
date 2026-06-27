package proxy

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/observability"
)

type Config struct {
	TargetBaseURL string
	Client        *http.Client
	Logger        *slog.Logger
	Telemetry     observability.TelemetryRecorder
}

func New(config Config) (http.Handler, error) {
	target, err := parseTargetBaseURL(config.TargetBaseURL)
	if err != nil {
		return nil, err
	}

	client := config.Client
	if client == nil {
		client = http.DefaultClient
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	mux.Handle("/", transparentProxy(target, client, config.Logger, config.Telemetry))
	return mux, nil
}

func parseTargetBaseURL(raw string) (*url.URL, error) {
	target, err := url.Parse(strings.TrimRight(raw, "/"))
	if err != nil {
		return nil, err
	}
	if target.Scheme == "" || target.Host == "" {
		return nil, errors.New("target base URL must include scheme and host")
	}
	return target, nil
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}
