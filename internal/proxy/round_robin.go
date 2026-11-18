package proxy

import (
	"net/url"
	"sync/atomic"
)

type roundRobin struct {
	targets []*url.URL
	next    atomic.Uint64
}

func newRoundRobin(targets []*url.URL) *roundRobin {
	return &roundRobin{
		targets: targets,
	}
}

func (rr *roundRobin) Next() *url.URL {
	index := (rr.next.Add(1) - 1) % uint64(len(rr.targets))
	return rr.targets[index]
}
