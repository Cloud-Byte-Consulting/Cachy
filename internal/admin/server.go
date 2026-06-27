package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/cloud-byte-consulting/cachy/internal/observability"
	"github.com/cloud-byte-consulting/cachy/internal/platform"
)

const Redacted = "<redacted>"

type ServerConfig struct {
	ListenAddress string
	Token         string
	Version       string
	Paths         platform.Paths
	RuntimeConfig RuntimeConfig
	Telemetry     TelemetrySnapshotter
}

type TelemetrySnapshotter interface {
	Snapshot() []observability.RequestTelemetry
}

type RuntimeConfig struct {
	ProxyListenAddress string
	TargetBaseURL      string
	ProviderCredential string
}

type StatusResponse struct {
	Status  string         `json:"status"`
	Version string         `json:"version"`
	Proxy   ProxyStatus    `json:"proxy"`
	Paths   platform.Paths `json:"paths"`
}

type ProxyStatus struct {
	ListenAddress    string `json:"listen_address"`
	TargetConfigured bool   `json:"target_configured"`
}

type ConfigResponse struct {
	ProxyListenAddress string `json:"proxy_listen_address"`
	TargetBaseURL      string `json:"target_base_url"`
	ProviderCredential string `json:"provider_credential,omitempty"`
}

type MetricsResponse struct {
	RequestsTotal     int            `json:"requests_total"`
	ErrorsTotal       int            `json:"errors_total"`
	AverageLatencyMS  int64          `json:"average_latency_ms"`
	StatusCounts      map[string]int `json:"status_counts"`
	FailureCategories map[string]int `json:"failure_categories"`
}

type DiagnosticsResponse struct {
	Health            string         `json:"health"`
	Paths             platform.Paths `json:"paths"`
	Ports             Ports          `json:"ports"`
	FailureCategories map[string]int `json:"failure_categories"`
}

type Ports struct {
	ProxyListenAddress string `json:"proxy_listen_address"`
	AdminListenAddress string `json:"admin_listen_address"`
}

type updateConfigRequest struct {
	TargetBaseURL string `json:"target_base_url"`
}

type server struct {
	mu      sync.RWMutex
	version string
	paths   platform.Paths
	config  RuntimeConfig
	admin   string

	telemetry TelemetrySnapshotter
}

func NewServer(config ServerConfig) (http.Handler, error) {
	if err := validateTargetBaseURL(config.RuntimeConfig.TargetBaseURL); err != nil {
		return nil, err
	}

	s := &server{
		version: config.Version,
		paths:   config.Paths,
		config:  config.RuntimeConfig,
		admin:   config.ListenAddress,

		telemetry: config.Telemetry,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/admin/v1/status", s.handleStatus)
	mux.HandleFunc("/admin/v1/config", s.handleConfig)
	mux.HandleFunc("/admin/v1/metrics", s.handleMetrics)
	mux.HandleFunc("/admin/v1/diagnostics", s.handleDiagnostics)

	return New(Config{
		ListenAddress: config.ListenAddress,
		Token:         config.Token,
	}, mux)
}

func (s *server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	writeJSON(w, http.StatusOK, metricsFromTelemetry(s.telemetrySnapshot()))
}

func (s *server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	s.mu.RLock()
	response := DiagnosticsResponse{
		Health: "ok",
		Paths:  s.paths,
		Ports: Ports{
			ProxyListenAddress: s.config.ProxyListenAddress,
			AdminListenAddress: s.admin,
		},
		FailureCategories: failureCategories(s.telemetrySnapshot()),
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, response)
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	writeJSON(w, http.StatusOK, StatusResponse{
		Status:  "ok",
		Version: s.version,
		Proxy: ProxyStatus{
			ListenAddress:    s.config.ProxyListenAddress,
			TargetConfigured: s.config.TargetBaseURL != "",
		},
		Paths: s.paths,
	})
}

func (s *server) telemetrySnapshot() []observability.RequestTelemetry {
	if s.telemetry == nil {
		return nil
	}
	return s.telemetry.Snapshot()
}

func (s *server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetConfig(w)
	case http.MethodPut:
		s.handlePutConfig(w, r)
	default:
		methodNotAllowed(w)
	}
}

func (s *server) handleGetConfig(w http.ResponseWriter) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	writeJSON(w, http.StatusOK, redactedConfigResponse(s.config))
}

func (s *server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var req updateConfigRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid config\n", http.StatusBadRequest)
		return
	}
	req.TargetBaseURL = strings.TrimRight(req.TargetBaseURL, "/")
	if err := validateTargetBaseURL(req.TargetBaseURL); err != nil {
		http.Error(w, "invalid target_base_url\n", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.config.TargetBaseURL = req.TargetBaseURL
	response := redactedConfigResponse(s.config)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, response)
}

func redactedConfigResponse(config RuntimeConfig) ConfigResponse {
	response := ConfigResponse{
		ProxyListenAddress: config.ProxyListenAddress,
		TargetBaseURL:      config.TargetBaseURL,
	}
	if config.ProviderCredential != "" {
		response.ProviderCredential = Redacted
	}
	return response
}

func validateTargetBaseURL(raw string) error {
	if raw == "" {
		return errors.New("target base URL is required")
	}
	target, err := url.Parse(raw)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return errors.New("target base URL must include scheme and host")
	}
	return nil
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed\n", http.StatusMethodNotAllowed)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func metricsFromTelemetry(records []observability.RequestTelemetry) MetricsResponse {
	response := MetricsResponse{
		StatusCounts:      map[string]int{},
		FailureCategories: map[string]int{},
	}
	var latencyTotal int64
	for _, record := range records {
		response.RequestsTotal++
		latencyTotal += record.Latency.Milliseconds()
		response.StatusCounts[strconv.Itoa(record.Status)]++
		if record.ErrorKind != "" && record.ErrorKind != observability.ErrorKindNone {
			response.ErrorsTotal++
			response.FailureCategories[string(record.ErrorKind)]++
		}
	}
	if response.RequestsTotal > 0 {
		response.AverageLatencyMS = latencyTotal / int64(response.RequestsTotal)
	}
	return response
}

func failureCategories(records []observability.RequestTelemetry) map[string]int {
	return metricsFromTelemetry(records).FailureCategories
}
