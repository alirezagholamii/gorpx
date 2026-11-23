package proxy

import (
	"net/url"
	"sync"
)

type roundRobin struct {
	mu      sync.Mutex
	targets []*url.URL
	health  []*upstreamHealth
	next    int
}

func newRoundRobin(targets []*url.URL) *roundRobin {
	rr := &roundRobin{
		targets: targets,
		health:  make([]*upstreamHealth, len(targets)),
	}

	for i := range targets {
		rr.health[i] = newUpstreamHealth()
	}

	return rr
}

func (rr *roundRobin) Next() (*url.URL, bool) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	count := len(rr.targets)

	if count == 0 {
		return nil, false
	}

	for i := 0; i < count; i++ {
		index := (rr.next + i) % count

		if rr.health[index].IsHealthy() {
			rr.next = (index + 1) % count
			return rr.targets[index], true
		}
	}

	return nil, false
}

func (rr *roundRobin) SetHealthy(index int, healthy bool) {
	rr.health[index].SetHealthy(healthy)
}
