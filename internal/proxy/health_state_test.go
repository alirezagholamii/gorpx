package proxy

import (
	"sync"
	"testing"
)

func TestUpstreamHealth(t *testing.T) {
	h := newUpstreamHealth()

	if !h.IsHealthy() {
		t.Fatal("new upstream should initially be healthy")
	}

	h.SetHealthy(false)

	if h.IsHealthy() {
		t.Fatal("upstream should be unhealthy")
	}

	h.SetHealthy(true)

	if !h.IsHealthy() {
		t.Fatal("upstream should be healthy again")
	}
}

func TestUpstreamHealthConcurrent(t *testing.T) {
	h := newUpstreamHealth()

	const goroutines = 100

	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			h.SetHealthy(i%2 == 0)
			_ = h.IsHealthy()
		}(i)
	}

	wg.Wait()
}
