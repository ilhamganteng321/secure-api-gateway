package gateway

import (
	"net"
	"net/http"
	"net/url"
	"testing"
)

func TestUpstreamPolicyRejectsUntrustedHost(t *testing.T) {
	policy := NewUpstreamPolicy([]string{
		"api.example.com",
	})

	err := policy.Validate("http://localhost:8080")

	if err == nil {
		t.Fatal("expected localhost to be rejected")
	}
}

func TestUpstreamPolicyAllowsConfiguredHost(t *testing.T) {
	policy := NewUpstreamPolicy([]string{
		"api.example.com",
	})

	err := policy.Validate("https://api.example.com")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"public", "8.8.8.8", true},
		{"private", "192.168.1.1", false},
		{"loopback", "127.0.0.1", false},
		{"unspecified", "0.0.0.0", false},
		{"link local", "169.254.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPublicIP(net.ParseIP(tt.ip))

			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeRequestHeaders(t *testing.T) {
	header := make(http.Header)

	header.Set("X-Forwarded-For", "127.0.0.1")
	header.Set("X-Original-URL", "/admin")
	header.Set("X-Custom", "safe")

	SanitizeRequestHeaders(header)

	if header.Get("X-Forwarded-For") != "" {
		t.Fatal("X-Forwarded-For should be removed")
	}

	if header.Get("X-Original-URL") != "" {
		t.Fatal("X-Original-URL should be removed")
	}

	if header.Get("X-Custom") != "safe" {
		t.Fatal("custom header should remain")
	}

	if header.Get("X-Gateway") != "secure-api-gateway" {
		t.Fatal("gateway header missing")
	}
}

func TestRejectURLUserInfo(t *testing.T) {
	policy := NewUpstreamPolicy([]string{
		"api.example.com",
	})

	u, _ := url.Parse("https://user:password@api.example.com")

	if err := policy.Validate(u.String()); err == nil {
		t.Fatal("expected URL userinfo to be rejected")
	}
}
