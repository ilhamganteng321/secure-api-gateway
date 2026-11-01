package gateway

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestUpstreamTimeout(t *testing.T) {
	fault := NewFaultServer()
	defer fault.Close()

	client := &http.Client{
		Timeout: 500 * time.Millisecond,
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fault.Server.URL+"/slow",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Do(req)

	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestUpstreamInternalError(t *testing.T) {
	fault := NewFaultServer()
	defer fault.Close()

	resp, err := http.Get(fault.Server.URL + "/error")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf(
			"expected status 500, got %d",
			resp.StatusCode,
		)
	}
}
