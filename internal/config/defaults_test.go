package config

import (
	"reflect"
	"sync"
	"testing"
)

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name       string
		server     Server
		wantHeader string
		wantIdle   string
	}{
		{
			name:       "both omitted",
			server:     Server{},
			wantHeader: "5s",
			wantIdle:   "60s",
		},
		{
			name: "custom values",
			server: Server{
				ReadHeaderTimeout: "10s",
				IdleTimeout:       "2m",
			},
			wantHeader: "10s",
			wantIdle:   "2m",
		},
		{
			name: "only header specified",
			server: Server{
				ReadHeaderTimeout: "15s",
			},
			wantHeader: "15s",
			wantIdle:   "60s",
		},
		{
			name: "only idle specified",
			server: Server{
				IdleTimeout: "30s",
			},
			wantHeader: "5s",
			wantIdle:   "30s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Server: tt.server,
			}

			result := ApplyDefaults(cfg)

			if result.Server.ReadHeaderTimeout != tt.wantHeader {
				t.Errorf(
					"ReadHeaderTimeout = %q, want %q",
					result.Server.ReadHeaderTimeout,
					tt.wantHeader,
				)
			}

			if result.Server.IdleTimeout != tt.wantIdle {
				t.Errorf(
					"IdleTimeout = %q, want %q",
					result.Server.IdleTimeout,
					tt.wantIdle,
				)
			}
		})
	}
}

func TestApplyRequestLimitDefaults(t *testing.T) {
	cfg := ApplyDefaults(Config{})

	if cfg.Server.MaxHeaderBytes != 16<<10 {
		t.Errorf(
			"MaxHeaderBytes = %d, want %d",
			cfg.Server.MaxHeaderBytes,
			16<<10,
		)
	}

	if cfg.Server.MaxBodyBytes != 10<<20 {
		t.Errorf(
			"MaxBodyBytes = %d, want %d",
			cfg.Server.MaxBodyBytes,
			10<<20,
		)
	}
}

func TestApplyCustomRequestLimits(t *testing.T) {
	cfg := Config{
		Server: Server{
			MaxHeaderBytes: 32 << 10,
			MaxBodyBytes:   20 << 20,
		},
	}

	result := ApplyDefaults(cfg)

	if result.Server.MaxHeaderBytes != 32<<10 {
		t.Errorf(
			"MaxHeaderBytes = %d, want %d",
			result.Server.MaxHeaderBytes,
			32<<10,
		)
	}

	if result.Server.MaxBodyBytes != 20<<20 {
		t.Errorf(
			"MaxBodyBytes = %d, want %d",
			result.Server.MaxBodyBytes,
			20<<20,
		)
	}
}

func TestApplyHealthCheckDefaults(t *testing.T) {
	health := &HealthCheck{Path: "/ready"}
	cfg := Config{
		Routes: []Route{
			{Path: "/default/", HealthCheck: &HealthCheck{}},
			{Path: "/custom/", HealthCheck: health},
			{Path: "/disabled/"},
		},
	}
	wantInput := Config{
		Routes: []Route{
			{Path: "/default/", HealthCheck: &HealthCheck{}},
			{Path: "/custom/", HealthCheck: &HealthCheck{Path: "/ready"}},
			{Path: "/disabled/"},
		},
	}

	got := ApplyDefaults(cfg)
	for i, want := range []HealthCheck{
		{Path: "/health", Interval: "10s", Timeout: "2s"},
		{Path: "/ready", Interval: "10s", Timeout: "2s"},
	} {
		if check := got.Routes[i].HealthCheck; check == nil || *check != want {
			t.Errorf("route %d health check = %+v, want %+v", i, check, want)
		}
	}
	if got.Routes[2].HealthCheck != nil {
		t.Error("ApplyDefaults enabled health checks on a route without them")
	}
	if !reflect.DeepEqual(cfg, wantInput) {
		t.Error("ApplyDefaults modified the input configuration")
	}
	if cfg.Routes[1].HealthCheck != health || *health != (HealthCheck{Path: "/ready"}) {
		t.Error("ApplyDefaults modified the original route entry or health check")
	}
	if &got.Routes[0] == &cfg.Routes[0] {
		t.Error("defaulted config still shares the original route slice")
	}
	if got.Routes[1].HealthCheck == health {
		t.Error("defaulted route still shares the original health check")
	}
	if twice := ApplyDefaults(got); !reflect.DeepEqual(got, twice) {
		t.Error("applying defaults twice changed the result")
	}
}

func TestApplyHealthCheckDefaultsConcurrently(t *testing.T) {
	health := &HealthCheck{}
	cfg := Config{Routes: []Route{{HealthCheck: health}}}
	var results [2]Config
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range results {
		wg.Go(func() {
			<-start
			results[i] = ApplyDefaults(cfg)
		})
	}
	close(start)
	wg.Wait()

	if !reflect.DeepEqual(results[0], results[1]) {
		t.Error("concurrent calls produced different configurations")
	}
	if cfg.Routes[0].HealthCheck != health || *health != (HealthCheck{}) {
		t.Error("concurrent calls modified the input route or health check")
	}
}
