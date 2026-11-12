package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr string
	}{
		{
			name: "valid configuration",
		},
		{
			name: "invalid server port",
			modify: func(cfg *Config) {
				cfg.Server.Address = ":99999"
			},
			wantErr: "invalid server port",
		},
		{
			name: "missing routes",
			modify: func(cfg *Config) {
				cfg.Routes = nil
			},
			wantErr: "at least one route",
		},
		{
			name: "duplicate route",
			modify: func(cfg *Config) {
				cfg.Routes = append(cfg.Routes, cfg.Routes[0])
			},
			wantErr: "duplicate route",
		},
		{
			name: "invalid upstream scheme",
			modify: func(cfg *Config) {
				cfg.Routes[0].Upstream = "ftp://localhost:3000"
			},
			wantErr: "invalid upstream",
		},
		{
			name: "missing destination",
			modify: func(cfg *Config) {
				cfg.Routes[0].Upstream = ""
			},
			wantErr: "exactly one",
		},
		{
			name: "both destinations",
			modify: func(cfg *Config) {
				cfg.Routes[0].Root = "./public"
			},
			wantErr: "exactly one",
		},
		{
			name: "empty server address",
			modify: func(cfg *Config) {
				cfg.Server.Address = ""
			},
			wantErr: "invalid server address",
		},
		{
			name: "missing upstream scheme",
			modify: func(cfg *Config) {
				cfg.Routes[0].Upstream = "localhost:3000"
			},
			wantErr: "invalid upstream",
		},
		{
			name: "empty upstream hostname",
			modify: func(cfg *Config) {
				cfg.Routes[0].Upstream = "http://"
			},
			wantErr: "invalid upstream",
		},
		{
			name: "invalid route prefix",
			modify: func(cfg *Config) {
				cfg.Routes[0].Path = "api/"
			},
			wantErr: "invalid route path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Server: Server{Address: ":8080"},
				Routes: []Route{
					{
						Path:     "/api/",
						Upstream: "http://localhost:3000",
					},
				},
			}

			if tt.modify != nil {
				tt.modify(&cfg)
			}

			err := Validate(cfg)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateStaticRoot(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		root    string
		wantErr bool
	}{
		{
			name: "valid directory",
			root: dir,
		},
		{
			name:    "missing directory",
			root:    filepath.Join(dir, "missing"),
			wantErr: true,
		},
		{
			name:    "regular file",
			root:    file,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Server: Server{Address: ":8080"},
				Routes: []Route{
					{
						Path: "/static/",
						Root: tt.root,
					},
				},
			}

			err := Validate(cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf(
					"Validate() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestValidateUpstream(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"HTTP", "http://localhost:3000", false},
		{"HTTPS", "https://example.com", false},
		{"IPv6", "http://[::1]:8080", false},
		{"default port", "https://example.com", false},
		{"port zero", "http://localhost:0", true},
		{"port too high", "http://localhost:99999", true},
		{"non-numeric port", "http://localhost:abc", true},
		{"empty port", "http://localhost:", true},
		{"unsupported scheme", "ftp://localhost:3000", true},
		{"credentials", "http://user:pass@localhost:3000", true},
		{"custom path", "http://localhost:3000/api", true},
		{"query", "http://localhost:3000?debug=true", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Server: Server{Address: ":8080"},
				Routes: []Route{
					{
						Path:     "/api/",
						Upstream: tt.url,
					},
				},
			}

			err := Validate(cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf(
					"Validate() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestValidateServerTimeouts(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{"valid seconds", "read_header", "5s", false},
		{"valid minutes", "idle", "1m", false},
		{"valid milliseconds", "read_header", "500ms", false},
		{"empty read timeout", "read_header", "", false},
		{"empty idle timeout", "idle", "", false},
		{"invalid duration", "read_header", "hello", true},
		{"missing unit", "idle", "5", true},
		{"zero duration", "read_header", "0s", true},
		{"negative duration", "idle", "-5s", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Server: Server{
					Address: ":8080",
				},
				Routes: []Route{
					{
						Path:     "/api/",
						Upstream: "http://localhost:3000",
					},
				},
			}

			switch tt.field {
			case "read_header":
				cfg.Server.ReadHeaderTimeout = tt.value
			case "idle":
				cfg.Server.IdleTimeout = tt.value
			}

			err := Validate(cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf(
					"Validate() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestValidateRequestLimits(t *testing.T) {
	tests := []struct {
		name       string
		headerSize int
		bodySize   int64
		wantErr    bool
	}{
		{
			name: "default values",
		},
		{
			name:       "custom values",
			headerSize: 32 << 10,
			bodySize:   20 << 20,
		},
		{
			name:       "negative header limit",
			headerSize: -1,
			wantErr:    true,
		},
		{
			name:     "negative body limit",
			bodySize: -1,
			wantErr:  true,
		},
		{
			name:       "both negative",
			headerSize: -1,
			bodySize:   -1,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Server: Server{
					Address:        ":8080",
					MaxHeaderBytes: tt.headerSize,
					MaxBodyBytes:   tt.bodySize,
				},
				Routes: []Route{
					{
						Path:     "/api/",
						Upstream: "http://localhost:3000",
					},
				},
			}

			cfg = ApplyDefaults(cfg)

			err := Validate(cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf(
					"Validate() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}
