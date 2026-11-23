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
		got, ok := rr.Next()
		if !ok {
			t.Fatal("expected a healthy upstream")
		}

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

			target, ok := rr.Next()
			if !ok {
				t.Error("expected a healthy upstream")
				return
			}

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

func TestRoundRobinSkipsUnhealthy(t *testing.T) {
	targets := []*url.URL{
		{Host: "backend-a"},
		{Host: "backend-b"},
		{Host: "backend-c"},
	}

	rr := newRoundRobin(targets)
	rr.SetHealthy(1, false)

	want := []string{
		"backend-a",
		"backend-c",
		"backend-a",
		"backend-c",
	}

	for i, expected := range want {
		target, ok := rr.Next()
		if !ok {
			t.Fatal("expected a healthy upstream")
		}

		if target.Host != expected {
			t.Errorf(
				"request %d: got %q, want %q",
				i+1,
				target.Host,
				expected,
			)
		}
	}
}

func TestRoundRobinAllUnhealthy(t *testing.T) {
	targets := []*url.URL{
		{Host: "backend-a"},
		{Host: "backend-b"},
	}

	rr := newRoundRobin(targets)
	rr.SetHealthy(0, false)
	rr.SetHealthy(1, false)

	target, ok := rr.Next()

	if ok || target != nil {
		t.Fatalf(
			"Next() = (%v, %v), want (nil, false)",
			target,
			ok,
		)
	}
}

func TestRoundRobinRecovery(t *testing.T) {
	targets := []*url.URL{
		{Host: "backend-a"},
		{Host: "backend-b"},
	}

	rr := newRoundRobin(targets)
	rr.SetHealthy(1, false)

	for i := 0; i < 4; i++ {
		target, ok := rr.Next()
		if !ok || target.Host != "backend-a" {
			t.Fatalf("expected backend-a while B is unhealthy")
		}
	}

	rr.SetHealthy(1, true)

	seen := make(map[string]bool)

	for i := 0; i < 2; i++ {
		target, ok := rr.Next()
		if !ok {
			t.Fatal("expected a healthy upstream")
		}

		seen[target.Host] = true
	}

	if !seen["backend-a"] || !seen["backend-b"] {
		t.Errorf("recovered backend was not selected: %v", seen)
	}
}

func TestRoundRobinConcurrentWithUnhealthy(t *testing.T) {
	targets := []*url.URL{
		{Host: "backend-a"},
		{Host: "backend-b"},
		{Host: "backend-c"},
	}

	rr := newRoundRobin(targets)

	// Exclude backend B from selection.
	rr.SetHealthy(1, false)

	const requests = 300

	counts := make(map[string]int)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < requests; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			target, ok := rr.Next()
			if !ok {
				t.Error("expected a healthy upstream")
				return
			}

			mu.Lock()
			counts[target.Host]++
			mu.Unlock()
		}()
	}

	wg.Wait()

	if counts["backend-a"] != 150 {
		t.Errorf(
			"backend-a received %d selections, want 150",
			counts["backend-a"],
		)
	}

	if counts["backend-b"] != 0 {
		t.Errorf(
			"unhealthy backend-b received %d selections",
			counts["backend-b"],
		)
	}

	if counts["backend-c"] != 150 {
		t.Errorf(
			"backend-c received %d selections, want 150",
			counts["backend-c"],
		)
	}
}
