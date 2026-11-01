package gateway

import (
	"errors"
	"sync/atomic"
)

var ErrNoHealthyUpstream = errors.New("no healthy upstream available")

type UpstreamInstance struct {
	Name    string
	URL     string
	Healthy atomic.Bool
}

type Balancer struct {
	instances []*UpstreamInstance
	counter   atomic.Uint64
}

func NewBalancer(instances []*UpstreamInstance) *Balancer {
	return &Balancer{
		instances: instances,
	}
}

func (b *Balancer) Next() (*UpstreamInstance, error) {
	if len(b.instances) == 0 {
		return nil, ErrNoHealthyUpstream
	}

	start := b.counter.Add(1)

	for i := 0; i < len(b.instances); i++ {
		index := int((start + uint64(i)) % uint64(len(b.instances)))
		instance := b.instances[index]

		if instance.Healthy.Load() {
			return instance, nil
		}
	}

	return nil, ErrNoHealthyUpstream
}
