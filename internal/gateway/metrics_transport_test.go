package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	registry := prometheus.NewRegistry()
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_upstream_requests_total"}, []string{"upstream", "status"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "test_upstream_duration_seconds"}, []string{"upstream"})
	errors := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_upstream_errors_total"}, []string{"upstream", "error_type"})
	registry.MustRegister(requests, duration, errors)

	transport := &MetricsTransport{
		Next:     http.DefaultTransport,
		Upstream: "users",
		Requests: requests,
		Duration: duration,
		Errors:   errors,
	}

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
