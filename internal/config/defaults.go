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

	return cfg
}
