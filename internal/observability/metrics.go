package observability

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	RequestsInFlight prometheus.Gauge
	Registry         *prometheus.Registry
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "route", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	requestsInFlight := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed.",
		},
	)

	registry.MustRegister(
		requestsTotal,
		requestDuration,
		requestsInFlight,
	)

	return &Metrics{
		RequestsTotal:    requestsTotal,
		RequestDuration:  requestDuration,
		RequestsInFlight: requestsInFlight,
		Registry:         registry,
	}
}

func (m *Metrics) Handler() fiber.Handler {
	return adaptor.HTTPHandler(
		promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}),
	)
}
