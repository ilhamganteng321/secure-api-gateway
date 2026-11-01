package gateway

import (
	"context"
	"net/http"
	"time"

	"github.com/ilhamganteng321/secure-api-gateway/internal/observability"
)

type HealthChecker struct {
	client  *http.Client
	metrics *observability.GatewayMetrics
}

func NewHealthChecker(timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (h *HealthChecker) SetGatewayMetrics(metrics *observability.GatewayMetrics) {
	h.metrics = metrics
}

func (h *HealthChecker) Check(
	ctx context.Context,
	upstream string,
	rawURL string,
) bool {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		rawURL,
		nil,
	)
	if err != nil {
		if h.metrics != nil {
			observability.RecordUpstreamHealth(h.metrics, upstream, false)
		}
		return false
	}

	resp, err := h.client.Do(req)
	if err != nil {
		if h.metrics != nil {
			observability.RecordUpstreamHealth(h.metrics, upstream, false)
		}
		return false
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300

	if h.metrics != nil {
		observability.RecordUpstreamHealth(h.metrics, upstream, healthy)
	}

	return healthy
}
