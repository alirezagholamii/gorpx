package proxy

import (
	"net/url"
	"sync"
)

type roundRobin struct {
	mu      sync.Mutex
	targets []*url.URL
	healthy []bool
	next    int
}

func newRoundRobin(targets []*url.URL) *roundRobin {
	rr := &roundRobin{
		targets: targets,
		healthy: make([]bool, len(targets)),
	}

	for i := range targets {
		rr.healthy[i] = true
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

		if rr.healthy[index] {
			rr.next = (index + 1) % count
			return rr.targets[index], true
		}
	}

	return nil, false
}

// SetHealthy reports whether the upstream's health changed.
func (rr *roundRobin) SetHealthy(index int, healthy bool) bool {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	changed := rr.healthy[index] != healthy
	rr.healthy[index] = healthy
	return changed
}
