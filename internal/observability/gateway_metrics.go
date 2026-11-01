package observability

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type GatewayMetrics struct {
	RequestsTotal *prometheus.CounterVec

	RequestDuration *prometheus.HistogramVec

	UpstreamRequestsTotal *prometheus.CounterVec

	UpstreamDuration *prometheus.HistogramVec

	UpstreamErrorsTotal *prometheus.CounterVec

	RateLimitRejections *prometheus.CounterVec

	CircuitBreakerState *prometheus.GaugeVec

	ActiveRequests prometheus.Gauge

	UpstreamHealthy *prometheus.GaugeVec
}

func NewGatewayMetrics(
	registry *prometheus.Registry,
) *GatewayMetrics {
	m := &GatewayMetrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_requests_total",
				Help: "Total gateway requests.",
			},
			[]string{"method", "route", "status"},
		),

		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gateway_request_duration_seconds",
				Help:    "Gateway request duration.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route"},
		),

		UpstreamRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_upstream_requests_total",
				Help: "Total upstream requests.",
			},
			[]string{"upstream", "status"},
		),

		UpstreamDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gateway_upstream_duration_seconds",
				Help:    "Upstream request duration.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"upstream"},
		),

		UpstreamErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_upstream_errors_total",
				Help: "Total upstream errors.",
			},
			[]string{"upstream", "error_type"},
		),

		RateLimitRejections: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_rate_limit_rejections_total",
				Help: "Total rate limit rejections.",
			},
			[]string{"client_id"},
		),

		CircuitBreakerState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_circuit_breaker_state",
				Help: "Circuit breaker state: 0=closed, 1=half-open, 2=open.",
			},
			[]string{"upstream"},
		),

		UpstreamHealthy: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_upstream_healthy",
				Help: "Upstream health status: 1=healthy, 0=unhealthy.",
			},
			[]string{"upstream"},
		),

		ActiveRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gateway_active_requests",
				Help: "Number of active gateway requests.",
			},
		),
	}

	registry.MustRegister(
		m.RequestsTotal,
		m.RequestDuration,
		m.UpstreamRequestsTotal,
		m.UpstreamDuration,
		m.UpstreamErrorsTotal,
		m.RateLimitRejections,
		m.CircuitBreakerState,
		m.UpstreamHealthy,
		m.ActiveRequests,
	)

	return m
}

func (m *GatewayMetrics) RecordRequest(
	method string,
	route string,
	status int,
	duration time.Duration,
) {
	m.RequestsTotal.WithLabelValues(
		method,
		route,
		strconv.Itoa(status),
	).Inc()

	m.RequestDuration.WithLabelValues(
		method,
		route,
	).Observe(duration.Seconds())
}

func RecordUpstreamRequest(
	metrics *GatewayMetrics,
	upstream string,
	status int,
	duration time.Duration,
) {
	if metrics == nil {
		return
	}

	metrics.UpstreamRequestsTotal.WithLabelValues(
		upstream,
		strconv.Itoa(status),
	).Inc()

	metrics.UpstreamDuration.WithLabelValues(
		upstream,
	).Observe(duration.Seconds())
}

func RecordCircuitBreakerState(
	metrics *GatewayMetrics,
	upstream string,
	state int,
) {
	if metrics == nil {
		return
	}

	metrics.CircuitBreakerState.
		WithLabelValues(upstream).
		Set(float64(state))
}

func (m *GatewayMetrics) RecordUpstreamHealth(
	upstream string,
	healthy bool,
) {
	if m == nil {
		return
	}

	value := 0.0
	if healthy {
		value = 1.0
	}

	m.UpstreamHealthy.WithLabelValues(upstream).Set(value)
}

func RecordUpstreamHealth(
	metrics *GatewayMetrics,
	upstream string,
	healthy bool,
) {
	if metrics == nil {
		return
	}

	metrics.RecordUpstreamHealth(upstream, healthy)
}
