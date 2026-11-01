package gateway_test

import (
	"sync"
	"testing"

	"github.com/ilhamganteng321/secure-api-gateway/internal/gateway"
)

func TestBalancerRoundRobin(t *testing.T) {
	instances := []*gateway.UpstreamInstance{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
	}
	for _, instance := range instances {
		instance.Healthy.Store(true)
	}

	balancer := gateway.NewBalancer(instances)

	got := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		instance, err := balancer.Next()
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, instance.Name)
	}

	seen := map[string]bool{}
	for _, name := range got {
		seen[name] = true
	}
	if len(seen) != 3 {
		t.Fatalf("expected all instances to be selected, got %v", got)
	}
}

func TestBalancerSkipsUnhealthyInstance(t *testing.T) {
	instances := []*gateway.UpstreamInstance{
		{Name: "a"},
		{Name: "b"},
	}
	instances[0].Healthy.Store(false)
	instances[1].Healthy.Store(true)

	balancer := gateway.NewBalancer(instances)

	for i := 0; i < 5; i++ {
		instance, err := balancer.Next()
		if err != nil {
			t.Fatal(err)
		}
		if instance.Name != "b" {
			t.Fatalf("expected healthy instance b, got %s", instance.Name)
		}
	}
}

func TestBalancerAllUnhealthy(t *testing.T) {
	instances := []*gateway.UpstreamInstance{
		{Name: "a"},
		{Name: "b"},
	}
	for _, instance := range instances {
		instance.Healthy.Store(false)
	}

	balancer := gateway.NewBalancer(instances)

	_, err := balancer.Next()
	if err != gateway.ErrNoHealthyUpstream {
		t.Fatalf("expected ErrNoHealthyUpstream, got %v", err)
	}
}

func TestBalancerConcurrent(t *testing.T) {
	instances := []*gateway.UpstreamInstance{{Name: "a"}}
	instances[0].Healthy.Store(true)

	balancer := gateway.NewBalancer(instances)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			instance, err := balancer.Next()
			if err != nil || instance.Name != "a" {
				t.Errorf("unexpected result: %v %v", instance, err)
			}
		}()
	}
	wg.Wait()
}
