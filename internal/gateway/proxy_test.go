package gateway_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/ilhamganteng321/secure-api-gateway/internal/gateway"
)

func TestReverseProxyForwardsRequest(t *testing.T) {
	upstreamServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/gateway/users/123" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}

			if r.Header.Get("X-Gateway-Upstream") != "users" {
				t.Error("missing gateway upstream header")
			}

			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"id":123,"name":"Test User"}`)
		},
	))
	defer upstreamServer.Close()

	upstream, err := gateway.NewUpstream(
		"users",
		upstreamServer.URL,
		2*time.Second,
	)
	if err != nil {
		t.Fatal(err)
	}

	proxy := gateway.NewProxy()

	if err := proxy.Register(upstream); err != nil {
		t.Fatal(err)
	}

	app := fiber.New()
	app.All("/gateway/users/*", proxy.Handler("users"))

	req := httptest.NewRequest(
		http.MethodGet,
		"/gateway/users/123",
		nil,
	)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestReverseProxyUpstreamTimeout(t *testing.T) {
	upstreamServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(300 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer upstreamServer.Close()

	upstream, err := gateway.NewUpstream(
		"slow-service",
		upstreamServer.URL,
		50*time.Millisecond,
	)
	if err != nil {
		t.Fatal(err)
	}

	proxy := gateway.NewProxy()

	if err := proxy.Register(upstream); err != nil {
		t.Fatal(err)
	}

	app := fiber.New()
	app.Get("/gateway/slow/*", proxy.Handler("slow-service"))

	req := httptest.NewRequest(
		http.MethodGet,
		"/gateway/slow/test",
		nil,
	)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf(
			"expected 504 Gateway Timeout, got %d",
			resp.StatusCode,
		)
	}
}

func TestUnknownUpstream(t *testing.T) {
	proxy := gateway.NewProxy()

	app := fiber.New()
	app.Get("/gateway/unknown/*", proxy.Handler("unknown"))

	req := httptest.NewRequest(
		http.MethodGet,
		"/gateway/unknown/test",
		nil,
	)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}
}
