package gateway

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

var ErrInvalidUpstream = errors.New("invalid upstream configuration")

type Upstream struct {
	Name    string
	BaseURL *url.URL
	Timeout time.Duration
}

func NewUpstream(
	name string,
	rawURL string,
	timeout time.Duration,
) (*Upstream, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrInvalidUpstream
	}

	if timeout <= 0 {
		return nil, ErrInvalidUpstream
	}

	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, ErrInvalidUpstream
	}

	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, ErrInvalidUpstream
	}

	if target.Host == "" ||
		target.User != nil ||
		target.RawQuery != "" ||
		target.Fragment != "" {
		return nil, ErrInvalidUpstream
	}

	return &Upstream{
		Name:    name,
		BaseURL: target,
		Timeout: timeout,
	}, nil
}
