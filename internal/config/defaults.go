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

	return cfg
}
