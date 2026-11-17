package proxy

import (
	"net/url"
	"sync"
	"testing"
)

func TestRoundRobin(t *testing.T) {
	targets := []*url.URL{
		{Host: "backend-a"},
		{Host: "backend-b"},
		{Host: "backend-c"},
	}

	rr := newRoundRobin(targets)

	want := []string{
		"backend-a",
		"backend-b",
		"backend-c",
		"backend-a",
		"backend-b",
		"backend-c",
	}

	for i, expected := range want {
		got := rr.Next()

		if got.Host != expected {
			t.Errorf(
				"request %d: got %q, want %q",
				i+1,
				got.Host,
				expected,
			)
		}
	}
}

func TestRoundRobinConcurrent(t *testing.T) {
	targets := []*url.URL{
		{Host: "backend-a"},
		{Host: "backend-b"},
		{Host: "backend-c"},
	}

	rr := newRoundRobin(targets)

	const requests = 300

	counts := make(map[string]int)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < requests; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			target := rr.Next()

			mu.Lock()
			counts[target.Host]++
			mu.Unlock()
		}()
	}

	wg.Wait()

	for _, target := range targets {
		if counts[target.Host] != 100 {
			t.Errorf(
				"%s received %d selections, want 100",
				target.Host,
				counts[target.Host],
			)
		}
	}
}
