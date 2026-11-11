package config

type Server struct {
	Address           string `yaml:"address"`
	ReadHeaderTimeout string `yaml:"read_header_timeout"`
	IdleTimeout       string `yaml:"idle_timeout"`
}

type Route struct {
	Path     string `yaml:"path"`
	Upstream string `yaml:"upstream"`
	Root     string `yaml:"root"`
}

type Config struct {
	Server Server  `yaml:"server"`
	Routes []Route `yaml:"routes"`
}

func Default() Config {
	return Config{
		Server: Server{
			Address:           ":8080",
			ReadHeaderTimeout: "5s",
			IdleTimeout:       "60s",
		},
	}
}
