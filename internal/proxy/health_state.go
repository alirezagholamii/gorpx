package proxy

import "sync/atomic"

type upstreamHealth struct {
	healthy atomic.Bool
}

func newUpstreamHealth() *upstreamHealth {
	h := &upstreamHealth{}
	h.healthy.Store(true)
	return h
}

func (h *upstreamHealth) IsHealthy() bool {
	return h.healthy.Load()
}

func (h *upstreamHealth) SetHealthy(healthy bool) {
	h.healthy.Store(healthy)
}
