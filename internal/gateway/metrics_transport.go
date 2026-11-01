package gateway

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ilhamganteng321/secure-api-gateway/internal/observability"
)

type MetricsTransport struct {
	Next     http.RoundTripper
	Upstream string

	// Direct Prometheus vectors, matching the Stage 11B example.
	Requests *prometheus.CounterVec
	Duration *prometheus.HistogramVec
	Errors   *prometheus.CounterVec

	// Optional convenience reference used by the application.
	Metrics *observability.GatewayMetrics
}

func (t *MetricsTransport) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	start := time.Now()

	resp, err := t.Next.RoundTrip(req)

	duration := time.Since(start)

	if err != nil {
		if t.Errors != nil {
			t.Errors.WithLabelValues(
				t.Upstream,
				"transport_error",
			).Inc()
		}
		if t.Metrics != nil {
			t.Metrics.UpstreamErrorsTotal.WithLabelValues(
				t.Upstream,
				"transport_error",
			).Inc()
		}

		return nil, err
	}

	status := http.StatusText(resp.StatusCode)
	if status == "" {
		status = "unknown"
	}

	if t.Requests != nil {
		t.Requests.WithLabelValues(
			t.Upstream,
			status,
		).Inc()
	}
	if t.Duration != nil {
		t.Duration.WithLabelValues(
			t.Upstream,
		).Observe(duration.Seconds())
	}
	if t.Errors != nil && resp.StatusCode >= 500 {
		t.Errors.WithLabelValues(
			t.Upstream,
			"http_5xx",
		).Inc()
	}

	if t.Metrics != nil {
		t.Metrics.UpstreamRequestsTotal.WithLabelValues(
			t.Upstream,
			status,
		).Inc()

		t.Metrics.UpstreamDuration.WithLabelValues(
			t.Upstream,
		).Observe(duration.Seconds())

		if resp.StatusCode >= 500 {
			t.Metrics.UpstreamErrorsTotal.WithLabelValues(
				t.Upstream,
				"http_5xx",
			).Inc()
		}
	}

	return resp, nil
}
