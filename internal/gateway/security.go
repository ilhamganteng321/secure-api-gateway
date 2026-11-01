package gateway

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
)

var ErrBlockedAddress = errors.New("upstream address is not allowed")

type UpstreamPolicy struct {
	AllowedHosts map[string]bool
}

func NewUpstreamPolicy(hosts []string) *UpstreamPolicy {
	allowed := make(map[string]bool, len(hosts))

	for _, host := range hosts {
		allowed[strings.ToLower(strings.TrimSpace(host))] = true
	}

	return &UpstreamPolicy{
		AllowedHosts: allowed,
	}
}

func (p *UpstreamPolicy) Validate(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ErrInvalidUpstream
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidUpstream
	}

	if u.Hostname() == "" || u.User != nil {
		return ErrInvalidUpstream
	}

	if u.RawQuery != "" || u.Fragment != "" {
		return ErrInvalidUpstream
	}

	host := strings.ToLower(u.Hostname())

	if !p.AllowedHosts[host] {
		return ErrBlockedAddress
	}

	return nil
}

func IsPublicIP(ip net.IP) bool {
	return ip != nil &&
		!ip.IsPrivate() &&
		!ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() &&
		!ip.IsUnspecified()
}

func ValidateResolvedHost(
	ctx context.Context,
	host string,
) error {
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return err
	}

	if len(ips) == 0 {
		return ErrBlockedAddress
	}

	for _, ip := range ips {
		if !IsPublicIP(ip) {
			return ErrBlockedAddress
		}
	}

	return nil
}
