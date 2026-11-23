package config

func ApplyDefaults(cfg Config) Config {
	defaults := Default()

	if cfg.Server.ReadHeaderTimeout == "" {
		cfg.Server.ReadHeaderTimeout =
			defaults.Server.ReadHeaderTimeout
	}

	if cfg.Server.IdleTimeout == "" {
		cfg.Server.IdleTimeout =
			defaults.Server.IdleTimeout
	}

	if cfg.Server.MaxHeaderBytes == 0 {
		cfg.Server.MaxHeaderBytes = defaults.Server.MaxHeaderBytes
	}

	if cfg.Server.MaxBodyBytes == 0 {
		cfg.Server.MaxBodyBytes = defaults.Server.MaxBodyBytes
	}

	for i := range cfg.Routes {
		hc := cfg.Routes[i].HealthCheck
		if hc == nil {
			continue
		}

		updated := *hc

		if updated.Path == "" {
			updated.Path = "/health"
		}
		if updated.Interval == "" {
			updated.Interval = "10s"
		}
		if updated.Timeout == "" {
			updated.Timeout = "2s"
		}

		cfg.Routes[i].HealthCheck = &updated
	}

	return cfg
}
