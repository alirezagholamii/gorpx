package config

import "testing"

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
