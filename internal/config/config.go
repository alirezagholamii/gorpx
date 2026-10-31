package config

type Server struct {
	Address string `yaml:"address"`
}

type Route struct {
	Path     string `yaml:"path"`
	Upstream string `yaml:"upstream"`
}

type Config struct {
	Server Server  `yaml:"server"`
	Routes []Route `yaml:"routes"`
}

func Default() Config {
	return Config{
		Server: Server{
			Address: ":8080",
		},
	}
}
