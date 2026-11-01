package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/sony/gobreaker/v2"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"

	"github.com/ilhamganteng321/secure-api-gateway/internal/observability"
)

type Proxy struct {
	upstreams map[string]*httputil.ReverseProxy
	breakers  map[string]*UpstreamBreaker
	metrics   *observability.GatewayMetrics
}

func NewProxy() *Proxy {
	return &Proxy{
		upstreams: make(map[string]*httputil.ReverseProxy),
		breakers:  make(map[string]*UpstreamBreaker),
	}
}

func (p *Proxy) SetGatewayMetrics(metrics *observability.GatewayMetrics) {
	p.metrics = metrics
}

func (p *Proxy) Register(upstream *Upstream) error {
	if upstream == nil || upstream.BaseURL == nil {
		return ErrInvalidUpstream
	}

	if _, exists := p.upstreams[upstream.Name]; exists {
		return errors.New("upstream already registered")
	}

	target := upstream.BaseURL

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,

		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,

		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: upstream.Timeout,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
	}

	p.breakers[upstream.Name] = NewUpstreamBreaker(upstream.Name)

	instrumentedTransport := otelhttp.NewTransport(transport)

	var finalTransport http.RoundTripper = instrumentedTransport
	if p.metrics != nil {
		finalTransport = &MetricsTransport{
			Next:     instrumentedTransport,
			Upstream: upstream.Name,
			Metrics:  p.metrics,
		}
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	reverseProxy.Transport = finalTransport

	originalDirector := reverseProxy.Director

	reverseProxy.Director = func(req *http.Request) {
		originalDirector(req)

		SanitizeRequestHeaders(req.Header)

		span := trace.SpanFromContext(req.Context())
		span.SetAttributes(
			attribute.String("gateway.upstream", upstream.Name),
		)

		req.Header.Set("X-Gateway-Upstream", upstream.Name)
	}

	reverseProxy.ErrorHandler = func(
		w http.ResponseWriter,
		r *http.Request,
		err error,
	) {
		status := http.StatusBadGateway
		message := "upstream service unavailable"

		var networkErr net.Error

		if errors.Is(err, context.DeadlineExceeded) ||
			(errors.As(err, &networkErr) && networkErr.Timeout()) {
			status = http.StatusGatewayTimeout
			message = "upstream request timeout"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)

		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": message,
		})
	}

	p.upstreams[upstream.Name] = reverseProxy

	return nil
}

func (p *Proxy) Handler(name string) fiber.Handler {
	reverseProxy, exists := p.upstreams[name]

	if !exists {
		return func(c fiber.Ctx) error {
			return fiber.NewError(
				fiber.StatusBadGateway,
				"upstream not configured",
			)
		}
	}

	breaker := p.breakers[name]

	return func(c fiber.Ctx) error {
		if p.metrics != nil && breaker != nil {
			state := 0
			switch breaker.State() {
			case gobreaker.StateHalfOpen:
				state = 1
			case gobreaker.StateOpen:
				state = 2
			}
			observability.RecordCircuitBreakerState(p.metrics, name, state)
		}

		return adaptor.HTTPHandler(reverseProxy)(c)
	}
}

// ParseTarget adalah helper untuk konfigurasi upstream.
func ParseTarget(raw string) (*url.URL, error) {
	return url.Parse(strings.TrimSpace(raw))
}
